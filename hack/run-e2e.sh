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

PROVIDER=${PROVIDER:-}
if [ "$PROVIDER" == "gke" ]; then
    KUBERNETES_SRC=
    KUBERNETES_PROVIDER=gke
    KUBERNETES_CONFORMANCE_PROVIDER=gke
    KUBE_GCE_ZONE=${GCP_ZONE:-}
    PROJECT=${GCP_PROJECT:-}
    DEFAULT_KUBECONFIG=$HOME/.kube/config
    KUBECTL=${KUBECTL:-kubectl}
else
    # legacy path
    source "${KUBE_ROOT}/cluster/common.sh"
    source "${KUBE_ROOT}/cluster/kube-util.sh"
    KUBERNETES_SRC=${KUBE_ROOT}
    KUBECTL=${KUBE_ROOT}/cluster/kubectl.sh
fi
KUBERNETES_PROVIDER=${KUBERNETES_PROVIDER:-} # e.g. local, gce
KUBERNETES_CONFORMANCE_TEST=${KUBERNETES_CONFORMANCE_TEST:-}
KUBERNETES_CONFORMANCE_PROVIDER=${KUBERNETES_CONFORMANCE_PROVIDER:-}
KUBE_GCE_ZONE=${KUBE_GCE_ZONE:-} # Available when provider is gce/gke
PROJECT=${PROJECT:-} # Available when provider is gce/gke
KUBECONFIG=${KUBECONFIG:-$DEFAULT_KUBECONFIG}
# In prow, ARTIFACTS environment indicates an existent directory where job
# artifacts can be dumped for automatic upload to GCS upon job completion.
ARTIFACTS=${ARTIFACTS:-}
PROVISIONER_E2E_IMAGE=${PROVISIONER_E2E_IMAGE:-}

echo "KUBERNETES_SRC: $KUBERNETES_SRC" >&2
echo "KUBERNETES_PROVIDER: $KUBERNETES_PROVIDER" >&2
echo "KUBERNETES_CONFORMANCE_PROVIDER: $KUBERNETES_CONFORMANCE_PROVIDER" >&2
echo "KUBERNETES_CONFORMANCE_TEST: $KUBERNETES_CONFORMANCE_TEST" >&2
echo "KUBE_GCE_ZONE: $KUBE_GCE_ZONE" >&2
echo "PROJECT: $PROJECT" >&2
echo "KUBECTL: $KUBECTL" >&2
echo "KUBECONFIG: $KUBECONFIG" >&2
echo "ARTIFACTS: $ARTIFACTS" >&2
echo "PROVISIONER_E2E_IMAGE: $PROVISIONER_E2E_IMAGE" >&2

if [ -z "$KUBERNETES_PROVIDER" -a -z "$KUBERNETES_CONFORMANCE_PROVIDER" ]; then
    echo "error: KUBERNETES_PROVIDER/KUBERNETES_CONFORMANCE_PROVIDER not set" >&2
    exit 1
fi

if [ -n "$KUBERNETES_CONFORMANCE_TEST" ]; then
    echo "Conformance test: not doing test setup."
else
    if [ "$PROVIDER" != "gke" ]; then
        echo "Setting up for KUBERNETES_PROVIDER=\"${KUBERNETES_PROVIDER}\"."
        prepare-e2e
        detect-master >/dev/null
    fi
fi

# build image if not specified
if [ -z "$PROVISIONER_E2E_IMAGE" ]; then
    make
    PROVISIONER_E2E_IMAGE=" k8s.gcr.io/sig-storage/local-volume-provisioner:latest_linux_amd64"
else
    docker pull $PROVISIONER_E2E_IMAGE
fi

# Why we use KUBERNETES_CONFORMANCE_PROVIDER here, see
# https://github.com/kubernetes/test-infra/blob/5475440d76f9039f7e1a5fa86c2f85ea8414b093/kubetest/gke.go#L210-L229.
if [ "$KUBERNETES_PROVIDER" == "gce" -o "$KUBERNETES_CONFORMANCE_PROVIDER" == "gke" ]; then
    if [ -z "$PROJECT" ]; then
        echo "info: PROJECT is not set, detect it automatically"
        PROJECT=$(gcloud config get-value project)
        if [ -z "$PROJECT" ]; then
            echo "error: PROJECT is required" >&2
            exit 1
        fi
    fi
    VERSION=$(git describe --tags --abbrev=8 --always)
    PROVISIONER_IMAGE_NAME=gcr.io/$PROJECT/local-volume-provisioner:$VERSION
    echo "Tag and push image $PROVISIONER_IMAGE_NAME"
    docker tag $PROVISIONER_E2E_IMAGE $PROVISIONER_IMAGE_NAME
    unset DOCKER_CONFIG # We don't need this and it may be read-only and fail the command to fail
    gcloud auth configure-docker
    docker push $PROVISIONER_IMAGE_NAME
    PROVISIONER_IMAGE_PULL_POLICY=Always
elif [ "$KUBERNETES_PROVIDER" == "local" ]; then
    KUBECONFIG=/var/run/kubernetes/admin.kubeconfig
    PROVISIONER_IMAGE_NAME=$PROVISIONER_E2E_IMAGE
    PROVISIONER_IMAGE_PULL_POLICY=Never
elif [ "$KUBERNETES_CONFORMANCE_PROVIDER" == "kind" ]; then
    PROVISIONER_IMAGE_NAME=$PROVISIONER_E2E_IMAGE
    kind load docker-image --name=kind-kubetest $PROVISIONER_IMAGE_NAME
    PROVISIONER_IMAGE_PULL_POLICY=Never
    # install sudo
    # FIXME: remove when we upgrade Kuberenetes code to 1.16+, see https://github.com/kubernetes/kubernetes/pull/80329.
    for n in $(kind get nodes --name=kind-kubetest); do
        docker exec $n sh -c 'apt-get update && apt-get install -y sudo'
    done
else
    echo "error: unsupported provider '$KUBERNETES_PROVIDER' or '$KUBERNETES_CONFORMANCE_PROVIDER'" >&2
    exit 1
fi

export PROVISIONER_IMAGE_NAME
export PROVISIONER_IMAGE_PULL_POLICY

TEST_ARGS=(
    test
    -timeout=60m
    -v
    ./test/e2e
)

if [ -n "$KUBECTL" ]; then
    TEST_ARGS+=("-kubectl-path=$KUBECTL")
fi

if [ -n "$KUBECONFIG" ]; then
    TEST_ARGS+=("-kubeconfig=$KUBECONFIG")
fi

if [ -n "$ARTIFACTS" ]; then
    TEST_ARGS+=("-report-dir=$ARTIFACTS")
fi

if [ -n "$KUBERNETES_PROVIDER" ]; then
    TEST_ARGS+=("-provider=$KUBERNETES_PROVIDER")
fi

if [ -n "$KUBE_GCE_ZONE" ]; then
    TEST_ARGS+=("-gce-zone=$KUBE_GCE_ZONE")
fi

TEST_ARGS+=("$@")

echo "Running e2e tests:" >&2
echo "PROVISIONER_IMAGE_NAME: $PROVISIONER_IMAGE_NAME"
echo "PROVISIONER_IMAGE_PULL_POLICY: $PROVISIONER_IMAGE_PULL_POLICY"
echo "go ${TEST_ARGS[@]}" >&2
exec "go" "${TEST_ARGS[@]}"
