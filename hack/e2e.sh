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
# Run `./hack/e2e.sh -h` to see help.

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "$ROOT/hack/lib.sh"

function usage() {
    cat <<'EOF'
This script is entrypoint to run e2e tests.

Usage: hack/e2e.sh [-h] -- [extra kubetest args]

    -h      show this message and exit

Environments:

    ARTIFACTS                           directory where job artifacts can be dumped
    PROVIDER                            local/gce/gke/skeleton (detect automatically if not specified)
    GCP_ZONE                            (for gce/gke) GCP zone
    GCP_PROJECT                         (for gce/gke) GCP project
    EXTRACT_STRATEGY                    kubetest extract strategy, see k8s.io/test-infra/kubetest/README.md for explanation
    KUBERNETES_SRC                      if specified, kubetest will skip extracting kubernetes src from GCS, use it instead
    DEPLOYMENT                          none/bash/gke/local/kind
    GKE_ENVIRONMENT                     (gke only) test/staging/prod
    GOOGLE_APPLICATION_CREDENTIALS      (for gce/gke) google applcation credentials which is used to access google cloud platform
    JENKINS_GCE_SSH_PRIVATE_KEY_FILE    (for gce/gke) GCP ssh key private file
    JENKINS_GCE_SSH_PUBLIC_KEY_FILE     (for gce/gke) GCP ssh key public file

Examples:

1) To run against existing local cluster started by k8s.io/kubernetes/hack/local-up-cluster.sh

    KUBERNETES_SRC=$GOPATH/src/k8s.io/kubernetes DEPLOYMENT=none ./hack/e2e.sh

  Optionally, you can add extra test args, e.g.

    KUBERNETES_SRC=$GOPATH/src/k8s.io/kubernetes DEPLOYMENT=none ./hack/e2e.sh -- --test-cmd-args=-ginkgo.focus='.*discovery.*'
    KUBERNETES_SRC=$GOPATH/src/k8s.io/kubernetes DEPLOYMENT=none ./hack/e2e.sh -- --test-cmd-args=-clean-start=true

2) To run against new local cluster started by k8s.io/kubernetes/hack/local-up-cluster.sh

    KUBERNETES_SRC=$GOPATH/src/k8s.io/kubernetes sudo -E env "PATH=$PATH" ./hack/e2e.sh

  Note that current kubetest needs root permission to cleanup.

3) To run against a cluster started by kind

    PROVIDER=skeleton DEPLOYMENT=kind ./hack/e2e.sh
    PROVIDER=skeleton DEPLOYMENT=kind KIND_NODE_IMAGE=kindest/node:v1.16.1 KUBERNETES_SRC=$GOPATH/src/k8s.io/kubernetes ./hack/e2e.sh

  WARNING: kind nodes share `/dev` with the host, so loop devices created in
  e2e tests can be seen on the host and may interfere with each other. It's not
  recommended to run this in shared environment.

4 ) To run against cluster with GCE provider locally

  You need install Google Cloud SDK first, then prepare google application
  credentials and configure ssh key pairs.

  You can create ssh keypair with ssh-keygen at  ~/.ssh/google_compute_engine
  or specifc existing ssh keypair with following environments:

    export JENKINS_GCE_SSH_PRIVATE_KEY_FILE=<path-to-your-ssh-private-key>
    export JENKINS_GCE_SSH_PUBLIC_KEY_FILE=<path-to-your-ssh-public-key>

  Then run with following environments:

    export GOOGLE_APPLICATION_CREDENTIALS=<path-to-your-google-application-credentials>
    export GCP_ZONE=<your-gcp-zone>
    export GCP_PROJECT=<your-gcp-project>
    ./hack/e2e.sh

5) To run against cluster with GKE provider locally

  Prepare same as with GCE provider. In addition, you need to grant Kubernetes
  Engine Admin (roles/container.admin) role to your GCP service account.

  Then run with following environments:

    export GOOGLE_APPLICATION_CREDENTIALS=<path-to-your-google-application-credentials>
    export GCP_ZONE=<your-gcp-zone>
    export GCP_PROJECT=<your-gcp-project>
    export PROVIDER=gke
    export GKE_ENVIRONMENT=prod
    ./hack/e2e.sh

6) To run against cluster with GCE/GKE provider in test-infra/prow job

  Almost same as running locally, you can add `preset-service-account:
  "true"` and `preset-k8s-ssh: "true"` labels in your prow job to use
  test-infra google application credentiails and GCP ssh key pair.

  The first label will set `GOOGLE_APPLICATION_CREDENTIALS` environment for
  you, and `kubetest` will acquire GCP project from boskos automatically. The
  latter will prepare SSH key pair.

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

ARTIFACTS=${ARTIFACTS:-}
PROVIDER=${PROVIDER:-}
GCP_ZONE=${GCP_ZONE:-us-central1-b}
GCP_PROJECT=${GCP_PROJECT:-}
EXTRACT_STRATEGY=${EXTRACT_STRATEGY:-ci/latest}
DEPLOYMENT=${DEPLOYMENT:-}
CLUSTER=${CLUSTER:-e2e}
GKE_ENVIRONMENT=${GKE_ENVIRONMENT:-prod}
KUBERNETES_SRC=${KUBERNETES_SRC:-} # If set, skip extracting kubernetes, use it as kubernetes src.
KIND_NODE_IMAGE=${KIND_NODE_IMAGE:-} # Prebuilt kind node image to use, e.g. kindest/node:v1.15.0.

if [ -z "$PROVIDER" ]; then
    echo "PROVIDER not specified, detecting provider automatically" >&2
    if [ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]; then
        echo "Found google application credentials at $GOOGLE_APPLICATION_CREDENTIALS, provider is set to gce" >&2
        PROVIDER=gce
    else
        PROVIDER=local
    fi
fi

echo "ARTIFACTS: $ARTIFACTS" >&2
echo "PROVIDER: $PROVIDER" >&2
echo "GCP_ZONE: $GCP_ZONE" >&2
echo "GCP_PROJECT: $GCP_PROJECT" >&2
echo "EXTRACT_STRATEGY: $EXTRACT_STRATEGY" >&2
echo "DEPLOYMENT: $DEPLOYMENT" >&2
echo "CLUSTER: $CLUSTER" >&2
echo "GKE_ENVIRONMENT: $GKE_ENVIRONMENT" >&2
echo "KUBERNETES_SRC: $KUBERNETES_SRC" >&2

kubetest_args=(
    --provider "$PROVIDER"
)

if [ -n "$ARTIFACTS" ]; then
    kubetest_args+=(
        --dump "${ARTIFACTS}"
    )
fi

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

if [ "$PROVIDER" == "gce" -o "$PROVIDER" == "gke" ]; then
    # GCP configurations for gce/gke
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

    if [ -n "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
        kubetest_args+=(
            --gcp-service-account "$GOOGLE_APPLICATION_CREDENTIALS"
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

    if [ "$PROVIDER" == "gce" ]; then
        if [ -z "$DEPLOYMENT" ]; then
            DEPLOYMENT=bash
        fi
    else
        # gke requires DEPLOYMENT=gke
        if [ -z "$DEPLOYMENT" ]; then
            DEPLOYMENT=gke
        fi
        if [ "$DEPLOYMENT" != "gke" ]; then
            echo "error: only DEPLOYMENT=gke is supported, found '$DEPLOYMENT'" >&2
            exit
        fi
        # gke required
        kubetest_args+=(
            --cluster "$CLUSTER"
            --gke-environment "$GKE_ENVIRONMENT"
            --gcp-node-image "cos"
            --gcp-network "e2e"
        )
    fi
elif [ "$PROVIDER" == "local" ]; then
    if [ -z "$DEPLOYMENT" ]; then
        DEPLOYMENT=local
    fi
elif [ "$PROVIDER" == "skeleton" ]; then
    if [ "$DEPLOYMENT" == "kind" ]; then
        export KUBERNETES_CONFORMANCE_PROVIDER=kind
        tmpfile=$(mktemp)
        trap "test -f $tmpfile && rm $tmpfile" EXIT
    cat <<EOF > $tmpfile
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
nodes:
- role: control-plane
- role: worker
- role: worker
EOF
        kubetest_args+=(
            --kind-config-path=$tmpfile
            --kind-binary-version=stable
        )
        if [ -n "$KIND_NODE_IMAGE" ]; then
            kubetest_args+=(
                --kind-node-image=$KIND_NODE_IMAGE
            )
        else
            kubetest_args+=(
                --build quick
            )
        fi
    fi
else
    echo "error: unsupported provider '$PROVIDER'" >&2
    exit 1
fi

if [ "${1:-}" == "--" ]; then
    shift
fi

if [ "$PROVIDER" == "gke" ]; then
    kubetest2_args+=(
        --up
        --down
        --test exec
        -v 1
        --cluster-name "$CLUSTER"
        --network "$CLUSTER"
        --gcp-service-account "$GOOGLE_APPLICATION_CREDENTIALS"
        --environment "$GKE_ENVIRONMENT"
    )
    if [ -n "$GCP_PROJECT" ]; then
        kubetest2_args+=(
            --project "$GCP_PROJECT"
        )
    fi
    if [ -n "$GCP_ZONE" ]; then
        kubetest2_args+=(
            --zone "$GCP_ZONE"
        )
    fi
    hack::install_kubetest2
    export PROVIDER
    export GCP_ZONE
    export GCP_PROJECT
    PATH=$OUTPUT_BIN:$PATH kubetest2-gke "${kubetest2_args[@]}" -- $ROOT/hack/run-e2e.sh "$@"
    exit
fi

# legacy path
go run $ROOT/hack/e2e.go -- "${kubetest_args[@]}" \
    --deployment "$DEPLOYMENT" \
    --up \
    --down \
    --test-cmd bash \
    --test-cmd-args="$ROOT/hack/run-e2e.sh" \
    "$@"
