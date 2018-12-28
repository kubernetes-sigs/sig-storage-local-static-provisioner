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

hack::install_dep

tmpdir=$(mktemp -d -t kirkops.vendor.XXXXXX)
trap "test -d $tmpdir && rm -rf $tmpdir" EXIT

echo "Backup vendor direcgtory to $tmpdir first"
mv vendor $tmpdir/vendor
cp Gopkg.lock $tmpdir/Gopkg.lock

$DEP_BIN ensure -v
diff -r --no-dereference --exclude "*.pyc" $tmpdir/vendor vendor && diff -u $tmpdir/Gopkg.lock Gopkg.lock
