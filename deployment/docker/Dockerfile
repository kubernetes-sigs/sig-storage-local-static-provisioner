# Copyright 2016 The Kubernetes Authors.
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

ARG OS=linux
ARG ARCH=amd64
ARG GOVERSION=1.11.1
FROM golang:$GOVERSION-alpine as builder
WORKDIR /go/src/sigs.k8s.io/sig-storage-local-static-provisioner
ADD . .
RUN CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} go build -a -ldflags '-extldflags "-static"' -o _output/${OS}/${ARCH}/local-volume-provisioner ./cmd/local-volume-provisioner

# For security, we use kubernetes community maintained debian base image.
# https://github.com/kubernetes/kubernetes/blob/master/build/debian-base/
FROM k8s.gcr.io/debian-base-${ARCH}:0.4.1

# Keep packages up to date and install packages for our needs.
RUN apt-get update \
    && apt-get upgrade -y \
    && clean-install \
    util-linux \
    e2fsprogs \
    bash

COPY --from=builder /go/src/sigs.k8s.io/sig-storage-local-static-provisioner/_output/${OS}/${ARCH}/local-volume-provisioner /local-provisioner
ADD deployment/docker/scripts /scripts
ENTRYPOINT ["/local-provisioner"]
