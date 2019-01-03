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

# This script is entrypoint to run e2e tests.
#
# It uses kubetest to setup/test/teardown kubernetes cluster.
#
# Examples:
#
# 1) To run against existing local cluster started by k8s.io/kubernetes/hack/local-up-cluster.sh
#
# KUBERNETES_SRC=$GOPATH/src/k8s.io/kubernetes DEPLOYMENT=none ./hack/e2e.sh
#
# Optionally, you can add extra test args, e.g.
#
# KUBERNETES_SRC=$GOPATH/src/k8s.io/kubernetes DEPLOYMENT=none ./hack/e2e.sh --test-cmd-args=-ginkgo.focus='.*discovery.*'
#
# 2) To run against new local cluster started by k8s.io/kubernetes/hack/local-up-cluster.sh
#
# KUBERNETES_SRC=$GOPATH/src/k8s.io/kubernetes sudo -E env "PATH=$PATH" ./hack/e2e.sh
#
# Note that current kubetest needs root permission to cleanup.
#
# 3) To run against cluster with GCE provider locally, specify following environments:
#
# export GOOGLE_APPLICATION_CREDENTIALS=<path-to-your-google-application-credentials>
# export GCP_ZONE=<gcp-zone>
# export GCP_PROJECT=<gcp-project>
#
# and create ssh keypair at ~/.ssh/google_compute_engine or specifc ssh keypair
# with following environments:
#
# export JENKINS_GCE_SSH_PRIVATE_KEY_FILE=<path-to-your-ssh-private-key>
# export JENKINS_GCE_SSH_PUBLIC_KEY_FILE=<path-to-your-ssh-public-key>
#
# 4) To run against cluster with GCE provider in test-infra/prow job, add
# `preset-service-account: "true"` and `preset-k8s-ssh: "true"` labels in your
# prow job.
#
# The first label will set `GOOGLE_APPLICATION_CREDENTIALS` environment for
# you, and `kubetest` will acquire GCP project and zone from boskos
# automatically. The latter will prepare SSH key pair.
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "$ROOT/hack/lib.sh"

PROVIDER=${PROVIDER:-}
GCP_ZONE=${GCP_ZONE:-}
GCP_PROJECT=${GCP_PROJECT:-}
EXTRACT_STRATEGY=${EXTRACT_STRATEGY:-ci/latest}
DEPLOYMENT=${DEPLOYMENT:-}
if [ -z "${KUBECTL:-}" ]; then
    KUBECTL=$(which kubectl 2>/dev/null || true)
fi
if [ -z "${KUBECTL:-}" ]; then
    echo "error: kubectl not found" >&2
    exit 1
fi
KUBERNETES_SRC=${KUBERNETES_SRC:-} # If set, skip extracting kubernetes, use it as kubernetes src.

if [ -z "$PROVIDER" ]; then
    echo "PROVIDER not specified, detecting provider automatically" >&2
    if [ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]; then
        echo "Found google application credentials at $GOOGLE_APPLICATION_CREDENTIALS, provider is set to gce" >&2
        PROVIDER=gce
    else
        PROVIDER=local
    fi
fi

echo "PROVIDER: $PROVIDER" >&2
echo "KUBECTL: $KUBECTL" >&2
echo "GCP_PROJECT: $GCP_PROJECT" >&2
echo "GCP_ZONE: $GCP_ZONE" >&2

kubetest_args=(
    --provider "$PROVIDER"
)

if [ -n "$KUBERNETES_SRC" ]; then
    echo "KUBERNETES_SRC is set to $KUBERNETES_SRC" >&2
    if [ ! -d "$KUBERNETES_SRC" ]; then
        echo "$KUBERNETES_SRC is not a directory" >&2
        exit 1
    fi
else
    kubetest_args+=(
        --extract  "$EXTRACT_STRATEGY"
    )
fi

if [ -n "$KUBERNETES_SRC" ]; then
    echo "KUBERNETES_SRC is set, entering into $KUBERNETES_SRC" >&2
    cd $KUBERNETES_SRC
fi

if [ "$PROVIDER" == "gce" ]; then
    if [ -n "$GCP_PROJECT" ]; then
        kubetest_args+=(
            --gcp-project "$GCP_PROJECT"
        )
    fi

    if [ -n "$GCP_ZONE" ]; then
        kubetest_args+=(
            --gcp-zone "$GCP_ZONE"
        )
    fi

    # kubetest needs ssh keypair to ssh into nodes
    if [ ! -d ~/.ssh ]; then
        mkdir ~/.ssh
    fi
    if [ -e ~/.ssh/google_compute_engine -o -n "$JENKINS_GCE_SSH_PRIVATE_KEY_FILE" ]; then
        echo "Copying $JENKINS_GCE_SSH_PRIVATE_KEY_FILE to ~/.ssh/google_compute_engine" >&2
        cp $JENKINS_GCE_SSH_PRIVATE_KEY_FILE ~/.ssh/google_compute_engine
        chmod 0600 ~/.ssh/google_compute_engine
    fi
    if [ -e ~/.ssh/google_compute_engine.pub -o -n "$JENKINS_GCE_SSH_PUBLIC_KEY_FILE" ]; then
        echo "Copying $JENKINS_GCE_SSH_PUBLIC_KEY_FILE to ~/.ssh/google_compute_engine.pub" >&2
        cp $JENKINS_GCE_SSH_PUBLIC_KEY_FILE ~/.ssh/google_compute_engine.pub
        chmod 0600 ~/.ssh/google_compute_engine.pub
    fi

    if [ -z "$DEPLOYMENT" ]; then
        DEPLOYMENT=bash
    fi
elif [ "$PROVIDER" == "local" ]; then
    if [ -z "$DEPLOYMENT" ]; then
        DEPLOYMENT=local
    fi
else
    echo "error: unsupported provider '$KUBERNETES_PROVIDER'" >&2
    exit 1
fi

go run $ROOT/hack/e2e.go -- "${kubetest_args[@]}" \
    --deployment "$DEPLOYMENT" \
    --up \
    --down \
    --test-cmd bash \
    --test-cmd-args="$ROOT/hack/run-e2e.sh" \
    "$@"
