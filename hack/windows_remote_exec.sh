#!/bin/bash

# Copyright 2021 The Kubernetes Authors.
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

set -ex

function get_windows_node() {
  echo $(kubectl get nodes -l kubernetes.io/os=windows -o jsonpath='{.items[*].metadata.name}')
}

function main() {
  echo "Compiling the test program"
  local output="_output/windows/amd64/vhd.exe"
  GO111MODULE=off CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o $output ./test/windows

  echo "Checking that the target windows node exists"
  local windows_node=$(get_windows_node)
  gcloud compute instances describe $windows_node > /dev/null

  echo "Executing the program remotely"
  local current_account=$(gcloud config list account --format "value(core.account)" | sed -r 's/@\S+//g')
  gcloud compute scp $output $windows_node:"C:\\Users\\${current_account}"
  gcloud compute ssh $windows_node --command="powershell -c .\vhd.exe"
}

main $@
