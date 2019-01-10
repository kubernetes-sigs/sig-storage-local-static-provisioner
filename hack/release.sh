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

# remove trailing `/` if present
REGISTRY=${REGISTRY%/}

GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

if [ -z "$ALLOW_DIRTY" -a "$GIT_DIRTY" != "clean" ]; then
    echo "error: repo status is not clean, skipped"
    exit 1
fi

# our logic depends repo tags, make sure all tags are fetched
echo "info: fetching all tags"
git fetch --tags

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

image="$REGISTRY/local-volume-provisioner:$VERSION"
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

# build & push container

if [ -z "$SKIP_BUILD" ]; then
    echo "info: building $image"
    pushd provisioner &>/dev/null
    make container REGISTRY=$REGISTRY VERSION=$VERSION
    popd
else
    echo "info: building is skipped"
fi

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

function is_latest_version() {
    local v="$1"
}

if ! is_stable_version "$VERSION"; then
    echo "error: VERSION '$VERSION' is not stable version, skipped pushing as latest image"
    exit 1
fi

latest_stable_version=$(git tag -l | grep -P '^v\d\.\d+\.\d+' | sort --version-sort | tail -n -1)
if [ -z "$latest_stable_version" ]; then
    echo "error: failed to get latest stable version"
    exit 1
fi

if [ "$VERSION" != "$latest_stable_version" ]; then
    echo "error: VERSION '$VERSION' is not latest stable version '$latest_stable_version', skiiped pushing as latest image"
    exit 1
fi

echo "info: VERSION '$VERSION' is latest stable version, pushing it as latest image"
latestimage="$REGISTRY/local-volume-provisioner:latest"
docker tag "$image" "$latestimage"
docker push "$latestimage"
