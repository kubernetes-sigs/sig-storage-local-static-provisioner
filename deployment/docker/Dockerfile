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

# For security, we use kubernetes community maintained debian base image.
# https://github.com/kubernetes/release/tree/master/images/build/debian-base
FROM registry.k8s.io/build-image/debian-base:bullseye-v1.2.0
ARG OS=linux
ARG ARCH=amd64
ARG binary=./_output/${OS}/${ARCH}/local-volume-provisioner
COPY ${binary} /local-provisioner

# Keep packages up to date and install packages for our needs.
RUN apt-get update \
    && apt-get upgrade -y \
    && clean-install \
    util-linux \
    e2fsprogs \
    bash

ADD deployment/docker/scripts /scripts
ENTRYPOINT ["/local-provisioner"]
