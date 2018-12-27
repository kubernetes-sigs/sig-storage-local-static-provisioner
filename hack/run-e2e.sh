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

# This script is executed by kubetest to run e2e tests.

set -o errexit
set -o nounset
set -o pipefail

# Note that kubetest run this script under kubernetes root directory.
KUBE_ROOT=$(pwd)
ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${KUBE_ROOT}/cluster/common.sh"
source "${KUBE_ROOT}/cluster/kube-util.sh"

KUBERNETES_SRC=${KUBE_ROOT}
KUBECTL=${KUBE_ROOT}/cluster/kubectl.sh
KUBERNETES_PROVIDER=${KUBERNETES_PROVIDER:-} # e.g. local, gce
KUBE_GCE_ZONE=${KUBE_GCE_ZONE:-} # Available when provider is gce
PROJECT=${PROJECT:-} # Available when provider is gce
KUBECONFIG=${KUBECONFIG:-$DEFAULT_KUBECONFIG}

echo "KUBERNETES_SRC: $KUBERNETES_SRC" >&2
echo "KUBERNETES_PROVIDER: $KUBERNETES_PROVIDER" >&2
echo "KUBE_GCE_ZONE: $KUBE_GCE_ZONE" >&2
echo "PROJECT: $PROJECT" >&2
echo "KUBECTL: $KUBECTL" >&2
echo "KUBECONFIG: $KUBECONFIG" >&2

if [ -z "$KUBERNETES_PROVIDER" ]; then
    echo "error: KUBERNETES_PROVIDER not set" >&2
    exit 1
fi

echo "Testing cluster with provider: ${KUBERNETES_PROVIDER}" >&2

prepare-e2e
detect-master >/dev/null

# build image
make

if [ "$KUBERNETES_PROVIDER" == "gce" ]; then
    if [ -z "$PROJECT" ]; then
        echo "error: PROJECT is required" >&2
        exit 1
    fi
    VERSION=$(git describe --tags --abbrev=8 --always)
    PROVISIONER_IMAGE_NAME=gcr.io/$PROJECT/local-volume-provisioner:$VERSION
    echo "Tag and push image $PROVISIONER_IMAGE_NAME"
    docker tag quay.io/external_storage/local-volume-provisioner:latest $PROVISIONER_IMAGE_NAME
    gcloud auth configure-docker
    docker push $PROVISIONER_IMAGE_NAME
    export PROVISIONER_IMAGE_NAME
    export PROVISIONER_IMAGE_PULL_POLICY=Always
elif [ "$KUBERNETES_PROVIDER" == "local" ]; then
    KUBECONFIG=/var/run/kubernetes/admin.kubeconfig
else
    echo "error: unsupported provider '$KUBERNETES_PROVIDER'" >&2
    exit 1
fi

TEST_ARGS=(
    test
    -timeout=60m
    -v
    github.com/kubernetes-sigs/sig-storage-local-static-provisioner/test/e2e
    "-provider=$KUBERNETES_PROVIDER"
)

if [ -n "$KUBECTL" ]; then
    TEST_ARGS+=("-kubectl-path=$KUBECTL")
fi

if [ -n "$KUBECONFIG" ]; then
    TEST_ARGS+=("-kubeconfig=$KUBECONFIG")
fi

TEST_ARGS+=("$@")

echo "Running e2e tests:" >&2
echo "go ${TEST_ARGS[@]}" >&2
exec "go" "${TEST_ARGS[@]}"
