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
    DOCKER_CONFIG               optional docker config location
    CONFIRM                     set this to skip confirmation
    ALLOW_UNSTABLE              by default, only master branch and tags that matches v<major>.<minor>.<patch> format are allowed, set this to skip this check (debug only)
    ALLOW_DIRTY                 by default, git repo must be clean, set this to skip this check (debug only)
    ALLOW_OVERRIDE              by default, stable image is not allowed to override, set this to skip (debug only)
    SKIP_BUILD                  set this to skip build phase (debug only)

Examples:

1) Release to your own registry for testing

    git tag v2.2.3
    REGISTRY=quay.io/<yourname> ./hack/release.sh

2) Release canary version

    REGISTRY=quay.io/<yourname> ALLOW_UNSTABLE=true VERSION=canary ./hack/release.sh

3) Release multi-arch image to your own registry

    REGISTRY=quay.io/<yourname> ALL_ARCH="amd64 arm64" ./hack/release.sh

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
CONFIRM=${CONFIRM:-}
DOCKER_CONFIG=${DOCKER_CONFIG:-}
ALLOW_UNSTABLE=${ALLOW_UNSTABLE:-}
ALLOW_DIRTY=${ALLOW_DIRTY:-}
ALLOW_OVERRIDE=${ALLOW_OVERRIDE:-}
SKIP_BUILD=${SKIP_BUILD:-}
# There is a problem in building multi-arch images in prow environment. Enable non-amd64
# arches when we have a reliable way, see https://github.com/kubernetes/test-infra/issues/13937.
# In the meantime, you may set the ALL_ARCH environment variable to name the architectures you'd
# like to target.
#ALL_ARCH="amd64 arm arm64 ppc64le s390x"
ALL_ARCH=${ALL_ARCH:-amd64}
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

GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

if [ -z "$ALLOW_DIRTY" -a "$GIT_DIRTY" != "clean" ]; then
    echo "error: repo status is not clean, skipped"
    exit 1
fi

# our logic depends repo tags, make sure all tags are fetched
echo "info: fetching all tags from official upstream"
git fetch --tags https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner.git

if [ -z "$VERSION" ]; then
    echo "info: VERSION is not specified, detect automatically"
    # get version from tag
    VERSION=$(git describe --tags --abbrev=0 --exact-match 2>/dev/null || true)
    if [ -z "$VERSION" ]; then
        echo "error: failed to detect version"
        exit 1
    fi
fi

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
    echo "info: building $image"
    for arch in $ALL_ARCH; do
        make provisioner REGISTRY=$REGISTRY VERSION=$VERSION ARCH=$arch
    done
else
    echo "info: building is skipped"
fi

function docker_push() {
    local image="$1"
    echo "info: pushing $image"
    docker_args=()
    if [ -n "$DOCKER_CONFIG" ]; then
        if [ ! -d "$DOCKER_CONFIG" ]; then
            echo "error: DOCKER_CONFIG '$DOCKER_CONFIG' does not exist or not a directory"
            exit 1
        fi
        if [ ! -f "$DOCKER_CONFIG/config.json" ]; then
            echo "error: docker config json '$DOCKER_CONFIG/config.json' does not exist"
            exit 1
        fi
        docker_args+=(--config "$DOCKER_CONFIG")
    fi
    docker_args+=(push "$image")
    docker "${docker_args[@]}"
}

for arch in $ALL_ARCH; do
    docker_push "$REGISTRY/local-volume-provisioner-$arch:$VERSION"
done

echo "info: create multi-arch manifest for $IMAGE:$VERSION"
function docker_create_multi_arch() {
    export DOCKER_CLI_EXPERIMENTAL=enabled
    local tag="$1"
    docker manifest create --amend $IMAGE:$tag $(echo ${ALL_ARCH} | sed -e "s~[^ ]*~${IMAGE}\-&:${tag}~g")
    for arch in $ALL_ARCH; do
        docker manifest annotate --arch ${arch} ${IMAGE}:${tag} ${IMAGE}-${arch}:${tag}
    done
    docker manifest push --purge $IMAGE:$tag
}

docker_create_multi_arch "$VERSION"

if ! is_stable_version "$VERSION"; then
    echo "info: VERSION '$VERSION' is not stable version, skip pushing as latest image"
    exit 0
fi

latest_stable_version=$(git tag -l | grep -P '^v\d\.\d+\.\d+$' | sort --version-sort | tail -n -1)
if [ -z "$latest_stable_version" ]; then
    echo "error: failed to get latest stable version"
    exit 1
fi

if [ "$VERSION" != "$latest_stable_version" ]; then
    echo "info: VERSION '$VERSION' is not latest stable version '$latest_stable_version', skip pushing as latest image"
    exit 0
fi

echo "info: VERSION '$VERSION' is latest stable version, push multi-arch images for latest tag"
for arch in $ALL_ARCH; do
    docker tag "$REGISTRY/local-volume-provisioner-$arch:$VERSION" "$REGISTRY/local-volume-provisioner-$arch:latest"
    docker_push "$REGISTRY/local-volume-provisioner-$arch:latest"
done

echo "info: VERSION '$VERSION' is latest stable version, create multi-arch manifest for latest tag"
docker_create_multi_arch "latest"
