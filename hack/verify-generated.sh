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

# lint first
ret=0
$HELM3_BIN lint ./provisioner || ret=$?
if [ $ret -ne 0 ]; then
    echo "helm lint failed"
    exit 2
fi

# check examples helm2
function test_values_helm2_file() {
    local input="examples/$1"
    local expected="generated_examples/helm2/$1"
    local tmpfile=$(mktemp)
    trap "test -f $tmpfile && rm $tmpfile || true" EXIT
    $HELM2_BIN template -f examples/$f --name local-static-provisioner --namespace default ./provisioner > $tmpfile
    echo -n "Checking $input "
    local diff=$(diff -u $expected $tmpfile 2>&1) || true
    if [[ -n "${diff}" ]]; then
        echo "failed, diff: "
        echo "$diff"
        exit 1
    else
        echo "passed."
    fi
}

# check examples helm3
function test_values_file() {
    local input="examples/$1"
    local expected="generated_examples/$1"
    local tmpfile=$(mktemp)
    trap "test -f $tmpfile && rm $tmpfile || true" EXIT
    $HELM3_BIN template --dry-run -f examples/$f local-static-provisioner --namespace default ./provisioner > $tmpfile
    echo -n "Checking $input "
    local diff=$(diff -u $expected $tmpfile 2>&1) || true
    if [[ -n "${diff}" ]]; then
        echo "failed, diff: "
        echo "$diff"
        exit 1
    else
        echo "passed."
    fi
}

FILES=$(ls examples/)
echo "==== HELM v2===="
for f in $FILES; do
    test_values_helm2_file $f
done
echo "==== HELM v3===="
for f in $FILES; do
    test_values_file $f
done
