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

REGISTRY ?= quay.io/external_storage
VERSION ?= latest
GOVERSION ?= 1.11.1
ARCH ?= amd64

ALL_ARCH = amd64 arm arm64 ppc64le s390x

IMAGE = $(REGISTRY)/local-volume-provisioner-$(ARCH):$(VERSION)
MUTABLE_IMAGE = $(REGISTRY)/local-volume-provisioner-$(ARCH):latest

TEMP_DIR := $(shell mktemp -d)
QEMUVERSION = v2.9.1

ifeq ($(ARCH),arm)
	QEMUARCH = arm
endif
ifeq ($(ARCH),arm64)
	QEMUARCH = aarch64
endif
ifeq ($(ARCH),ppc64le)
	QEMUARCH = ppc64le
endif
ifeq ($(ARCH),s390x)
	QEMUARCH = s390x
endif
   
SUDO = $(if $(filter 0,$(shell id -u)),,sudo)

all: cross
.PHONY: all

cross: $(addprefix provisioner-,$(ALL_ARCH))
.PHONY: cross

verify:
	./hack/verify-all.sh
.PHONY: verify

e2e:
	./hack/e2e.sh
.PHONY: e2e

release:
	./hack/release.sh
.PHONY: release

provisioner-%: 
	$(MAKE) ARCH=$* provisioner

provisioner:
	mkdir -p _output
	# because COPY does not expand build arguments, we need substitute it
	cat ./deployment/docker/Dockerfile \
		| sed "s|QEMUARCH|$(QEMUARCH)|g" \
		> $(TEMP_DIR)/Dockerfile
ifeq ($(ARCH),amd64)
	# When building "normally" for amd64, remove the whole line, it has no part in the amd64 image
	sed "/CROSS_BUILD_/d" $(TEMP_DIR)/Dockerfile > $(TEMP_DIR)/Dockerfile.tmp
else
	# When cross-building, only the placeholder "CROSS_BUILD_" should be removed
	# Register /usr/bin/qemu-ARCH-static as the handler for non-x86 binaries in the kernel
	$(SUDO) ./third_party/multiarch/qemu-user-static/register/register.sh --reset
	curl -sSL https://github.com/multiarch/qemu-user-static/releases/download/$(QEMUVERSION)/x86_64_qemu-$(QEMUARCH)-static.tar.gz | tar -xz -C _output
	# Ensure we don't get surprised by umask settings
	chmod 0755 _output/qemu-$(QEMUARCH)-static
	sed "s/CROSS_BUILD_//g" $(TEMP_DIR)/Dockerfile > $(TEMP_DIR)/Dockerfile.tmp
endif
	mv $(TEMP_DIR)/Dockerfile.tmp $(TEMP_DIR)/Dockerfile
	docker build -t $(MUTABLE_IMAGE) --build-arg GOVERSION=$(GOVERSION) --build-arg ARCH=$(ARCH) -f $(TEMP_DIR)/Dockerfile .
	docker tag $(MUTABLE_IMAGE) $(IMAGE)
	rm -rf $(TEMP_DIR)
.PHONY: provisioner

test: provisioner
	go test ./cmd/... ./pkg/...
	docker run --privileged -v $(PWD)/deployment/docker/test.sh:/test.sh --entrypoint bash $(IMAGE) /test.sh
.PHONY: test

clean:
	rm -rf _output
.PHONY: clean
