#!/bin/bash

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script is entrypoint to release images automatically.
#
# Run `./hack/release.sh -h` to see help.

set -o errexit
set -o nounset
set -o pipefail
set -x

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "$ROOT/hack/lib.sh"

function usage() {
    cat <<'EOF'
This script is entrypoint to release images automatically.

Note that this script expected

Usage: hack/release.sh

    -h      show this message and exit

Environments:

    REGISTRY                    container registry without repo name (default: quay.io/external_storage)
    VERSION                     if set, use given version as image tag
    PULL_BASE_REF               if set, detect version from this git ref instead of "git describe"
    DOCKER_CONFIG               optional docker config location
    CONFIRM                     set this to skip confirmation
    ALLOW_UNSTABLE              by default, only master branch and tags that matches v<major>.<minor>.<patch> format are allowed, set this to skip this check (debug only)
    ALLOW_DIRTY                 by default, git repo must be clean, set this to skip this check (debug only)
    ALLOW_OVERRIDE              by default, stable image is not allowed to override, set this to skip (debug only)
    SKIP_BUILD                  set this to skip build phase (debug only)
    SKIP_PUSH_LATEST            set this to skip pushing the latest stable image as the latest image
    LINUX_ARCH                  Linux architectures to build
    WINDOWS_DISTROS             Windows distros to build

Examples:

1) Release to your own registry for testing

    git tag v2.2.3
    REGISTRY=quay.io/<yourname> ./hack/release.sh

2) Release canary version

    REGISTRY=quay.io/<yourname> ALLOW_UNSTABLE=true VERSION=canary ./hack/release.sh

3) Release multi-arch image to your own registry

    REGISTRY=quay.io/<yourname> LINUX_ARCH="amd64 arm64" WINDOWS_DISTROS="ltsc2019 1909" ./hack/release.sh

EOF
}

while getopts "h?" opt; do
    case "$opt" in
    h|\?)
        usage
        exit 0
        ;;
    esac
done

REGISTRY=${REGISTRY:-quay.io/external_storage}
VERSION=${VERSION:-}
PULL_BASE_REF=${PULL_BASE_REF:-}
CONFIRM=${CONFIRM:-}
DOCKER_CONFIG=${DOCKER_CONFIG:-}
ALLOW_UNSTABLE=${ALLOW_UNSTABLE:-}
ALLOW_DIRTY=${ALLOW_DIRTY:-}
ALLOW_OVERRIDE=${ALLOW_OVERRIDE:-}
SKIP_BUILD=${SKIP_BUILD:-}
SKIP_PUSH_LATEST=${SKIP_PUSH_LATEST:-}
LINUX_ARCH=${LINUX_ARCH:-amd64 arm arm64 ppc64le s390x}
WINDOWS_DISTROS=${WINDOWS_DISTROS:-ltsc2019 1909 2004 20H2}

echo "REGISTRY: $REGISTRY"
echo "VERSION: $VERSION"
echo "PULL_BASE_REF: $PULL_BASE_REF"
echo "CONFIRM: $CONFIRM"
echo "DOCKER_CONFIG: $DOCKER_CONFIG"
echo "ALLOW_UNSTABLE: $ALLOW_UNSTABLE"
echo "ALLOW_DIRTY: $ALLOW_DIRTY"
echo "ALLOW_OVERRIDE: $ALLOW_OVERRIDE"
echo "SKIP_BUILD: $SKIP_BUILD"
echo "SKIP_PUSH_LATEST: $SKIP_PUSH_LATEST"
echo "LINUX_ARCH: $LINUX_ARCH"
echo "WINDOWS_DISTROS: $WINDOWS_DISTROS"

IMAGE="$REGISTRY/local-volume-provisioner"

# In prow job, DOCKER_CONFIG is mounted read-only, but docker manifest command
# expects it is writable.
if [ -n "$DOCKER_CONFIG" ]; then
    tmpDir=$(mktemp -d)
    echo "info: copy $DOCKER_CONFIG/config.json to $tmpDir and set DOCKER_CONFIG to $tmpDir"
    cp -L $DOCKER_CONFIG/config.json $tmpDir/
    DOCKER_CONFIG=$tmpDir
fi

# remove trailing `/` if present
REGISTRY=${REGISTRY%/}

if [ -z "$ALLOW_DIRTY" ]; then
    GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "dirty" || echo "clean")
    if [ "$GIT_DIRTY" != "clean" ]; then
        echo "error: repo status is not clean, skipped"
        exit 1
    fi
fi

if [ -z "$VERSION" ]; then
    echo "info: VERSION is not specified, detect automatically"
    if [ -n "$PULL_BASE_REF" ]; then
        echo "info: detecting version from PULL_BASE_REF '$PULL_BASE_REF'"
        if [[ "$PULL_BASE_REF" == "master" ]]; then
            VERSION=canary
        elif [[ "$PULL_BASE_REF" =~ release-\d+ ]]; then
            # release-2.0
            VERSION=$(echo "$PULL_BASE_REF" | cut -f2 -d '-')-canary
        elif [[ "$PULL_BASE_REF" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$ ]]; then
            # stable release: v1.2.3
            # or pre-release: v1.2.3-rc1
            VERSION="$PULL_BASE_REF"
        fi
    else
        echo "info: detecting version from 'git describe'"
        VERSION=$(git describe --tags --abbrev=0 --exact-match 2>/dev/null || true)
    fi
    if [ -z "$VERSION" ]; then
        echo "error: failed to detect version"
        exit 1
    fi
fi

echo "info: VERSION is $VERSION and will be used as image tag"

# By default, only v<major>.<minor>.<patch> version are allowed.
function is_stable_version() {
    [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

if [ -z "$ALLOW_UNSTABLE" ]; then
    if ! is_stable_version "$VERSION"; then
        echo "error: unstable version '$VERSION', must match regex 'v[0-9]+\.[0-9]+\.[0-9]+$' or skip this check with ALLOW_UNSTABLE"
        exit 1
    fi
fi

image="$IMAGE:$VERSION"
if [ -z "$CONFIRM" ]; then
    read -r -p "info: build and push to $image? [y/N]" response
    if [[ ! $response =~ ^([yY][eE][sS]|[yY])$ ]]; then
        echo "Exited."
        exit 0
    fi
fi

# Usage: manifest_url <domain>/<name>:<reference>
# xref: https://docs.docker.com/registry/spec/api/
function get_manifest_url() {
    read -r domain namewithref <<<"${1/\// }"
    read -r name ref <<<"${namewithref/:/ }"
    echo "https://${domain}/v2/${name}/manifests/${ref}"
}

if [ -z "$ALLOW_OVERRIDE" ] && is_stable_version "$VERSION"; then
    echo "info: $image is stable image, checking if it does eixst"
    manifest_url=$(get_manifest_url "$image")
    code=$(curl -s -XHEAD -w '%{http_code}' "$manifest_url" || true)
    if [ "$code" == "200" ]; then
        echo "error: $image does exist, skipped"
        exit 1
    elif [ "$code" != "404" ]; then
        echo "error: unexpected http code '$code'"
        exit 1
    fi
    echo "info: '$image' does not exist, continue"
fi

# build & push multi-arch images
if [ -z "$SKIP_BUILD" ]; then
    echo "info: build and push $image"
    make cross \
        REGISTRY=$REGISTRY \
        VERSION=$VERSION \
        LINUX_ARCH="$LINUX_ARCH" \
        WINDOWS_DISTROS="$WINDOWS_DISTROS"
else
    echo "info: build and push is skipped"
fi

echo "info: create multi-arch manifest for $IMAGE:$VERSION"
function docker_create_multi_arch() {
    export DOCKER_CLI_EXPERIMENTAL=enabled

    # tag_version is the version used in the docker manifest that ties
    # all of the images that were tagged with $VERSION
    local tag_version=${1}
    local manifest_image=$IMAGE:$tag_version

    # get the list of all the images created
    local linux_images=$(echo "${LINUX_ARCH}" | tr ' ' '\n' | while read -r arch; do \
        echo $IMAGE:${VERSION}_linux_${arch}; \
    done);
    local windows_images=$(echo "${WINDOWS_DISTROS}" | tr ' ' '\n' | while read -r distro; do \
        echo $IMAGE:${VERSION}_windows_${distro}; \
    done);
    local all_images="${linux_images} ${windows_images}"

    # create a manifest with all the images created
    docker manifest create --amend $manifest_image $all_images

    # annotate the linux images with the right arch
    # from https://github.com/kubernetes/release/blob/8dbca63a6875e59e2234954ad3876d9490bbeede/images/build/debian-base/Makefile#L67-L70
    echo "${LINUX_ARCH}" | tr ' ' '\n' | while read -r arch; do
        local linux_image=$IMAGE:${VERSION}_linux_${arch}
        docker manifest annotate --arch $arch $manifest_image $linux_image
    done

    # annotate the windows images with the base image os-version
    # from https://github.com/kubernetes-csi/csi-release-tools/blob/5b9a1e06794ddb137ff7e2d565416cc6934ec380/build.make#L181-L189
    echo "${WINDOWS_DISTROS}" | tr ' ' '\n' | while read -r distro; do
        local windows_image=$IMAGE:${VERSION}_windows_${distro}
        # the image matches the value in the Makefile
        local os_version=$(docker manifest inspect mcr.microsoft.com/windows/servercore:${distro} | grep "os.version" | head -n 1 | awk '{print $2}' | sed -e 's/"//g')
        docker manifest annotate --os-version ${os_version} $manifest_image $windows_image
    done

    docker manifest push --purge $manifest_image
}

docker_create_multi_arch $VERSION

if ! is_stable_version "$VERSION" || [ -n "$SKIP_PUSH_LATEST" ]; then
    echo "info: VERSION '$VERSION' is not stable version or SKIP_PUSH_LATEST is set, skip pushing $VERSION as the latest image"
    exit 0
fi

echo "info: fetching all tags from official upstream"
git fetch --tags https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner.git

latest_stable_version=$(git tag -l | grep -P '^v\d\.\d+\.\d+$' | sort --version-sort | tail -n -1)
if [ -z "$latest_stable_version" ]; then
    echo "error: failed to get latest stable version"
    exit 1
fi

if [ "$VERSION" != "$latest_stable_version" ]; then
    echo "info: VERSION '$VERSION' is not latest stable version '$latest_stable_version', skip pushing as latest image"
    exit 0
fi

echo "info: VERSION '$VERSION' is latest stable version, tagging $IMAGE as latest"
docker_create_multi_arch latest
