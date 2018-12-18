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

function hack::install_helm() {
    local OS=$(uname | tr A-Z a-z)
    local VERSION=v2.7.2
    local ARCH=amd64
    local HELM_URL=http://storage.googleapis.com/kubernetes-helm/helm-${VERSION}-${OS}-${ARCH}.tar.gz
    curl -s "$HELM_URL" | sudo tar --strip-components 1 -C $GOPATH/bin -zxf - ${OS}-${ARCH}/helm
}
