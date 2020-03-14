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

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

hack::install_helm2
hack::install_helm3

cd helm

echo "*** IMPORTANT NOTE ***"
cat <<EOF
This script is used to update generated yaml files from helm values
automatically. It does not validate generated yaml files. Please check to make
sure generated files are what you expected.
EOF
echo "*** IMPORTANT NOTE ***"

echo "====== HELM v2 ========"

FILES=$(ls examples/)
for f in $FILES; do
    input="examples/$f"
    generated="generated_examples/helm2/$f"
    printf "Generating with helm v${HELM2_VERSION} %s from %s\n" $generated $input
    $HELM2_BIN template -f examples/$f  --name local-static-provisioner --namespace default ./provisioner > $generated
done

echo "====== HELM v3 ========"

FILES=$(ls examples/)
for f in $FILES; do
    input="examples/$f"
    generated="generated_examples/$f"
    printf "Generating with helm v${HELM3_VERSION} %s from %s\n" $generated $input
    $HELM3_BIN template --dry-run -f examples/$f local-static-provisioner --namespace default ./provisioner > $generated
done

echo "Done."
