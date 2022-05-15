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

REGISTRY ?= registry.k8s.io/sig-storage
VERSION ?= latest
GOVERSION ?= 1.17

# These env vars have default values set from hack/release.sh, the values
# shown here are for `make` and `make verify` only
LINUX_ARCH ?= amd64
WINDOWS_DISTROS ?=

WINDOWS_BASE_IMAGES=$(addprefix mcr.microsoft.com/windows/nanoserver:,$(WINDOWS_DISTROS))

DOCKER=DOCKER_CLI_EXPERIMENTAL=enabled docker
STAGINGVERSION=${VERSION}
STAGINGIMAGE=${REGISTRY}/local-volume-provisioner
# Output type of docker buildx build
OUTPUT_TYPE ?= docker

# $(call pos,slice,wanted)
# finds the index of `wanted` in `slice`
_pos = $(if $(findstring $1,$2),$(call _pos,$1,\
       $(wordlist 2,$(words $2),$2),x $3),$3)
pos = $(words $(call _pos,$1,$2))

# $(call lookup,wanted,list1,list2)
# finds the index of `wanted` in list1, then, it returns the element of `list2`
# at that index
lookup = $(word $(call pos,$1,$2),$3)

all: build-container-linux-amd64
.PHONY: all

cross: init-buildx \
	$(addprefix build-and-push-container-linux-,$(LINUX_ARCH)) \
	$(addprefix build-and-push-container-windows-,$(WINDOWS_DISTROS))
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

# used in `make test` and `make e2e`
# builds without pushing to the registry
build-container-linux-%:
	CGO_ENABLED=0 GOOS=linux GOARCH=$* go build -a -ldflags '-extldflags "-static"' -mod vendor -o _output/linux/$*/local-volume-provisioner ./cmd/local-volume-provisioner
	$(DOCKER) buildx build --file=./deployment/docker/Dockerfile --platform=linux/$* \
		-t $(STAGINGIMAGE):$(STAGINGVERSION)_linux_$* --output=type=$(OUTPUT_TYPE) \
		--build-arg OS=linux \
		--build-arg ARCH=$* .

build-and-push-container-linux-%: init-buildx
	CGO_ENABLED=0 GOOS=linux GOARCH=$* go build -a -ldflags '-extldflags "-static"' -mod vendor -o _output/linux/$*/local-volume-provisioner ./cmd/local-volume-provisioner
	$(DOCKER) buildx build --file=./deployment/docker/Dockerfile --platform=linux/$* \
		-t $(STAGINGIMAGE):$(STAGINGVERSION)_linux_$* \
		--build-arg OS=linux \
		--build-arg ARCH=$* \
		--push .

build-and-push-container-windows-%: init-buildx
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -ldflags='-extldflags="-static" -X="main.version=${STAGINGVERSION}"' -mod vendor -o _output/windows/amd64/local-volume-provisioner.exe ./cmd/local-volume-provisioner
	$(DOCKER) buildx build --file=./deployment/docker/Dockerfile.Windows --platform=windows/amd64 \
		-t $(STAGINGIMAGE):$(STAGINGVERSION)_windows_$* \
		--build-arg BASE_IMAGE=$(call lookup,$*,$(WINDOWS_DISTROS),$(WINDOWS_BASE_IMAGES)) \
		--push .

test: build-container-linux-amd64
	go test ./cmd/... ./pkg/...
	docker run --privileged -v $(PWD)/deployment/docker/test.sh:/test.sh --entrypoint bash $(STAGINGIMAGE):$(STAGINGVERSION)_linux_amd64 /test.sh
.PHONY: test

clean:
	rm -rf _output
.PHONY: clean

init-buildx:
	# Ensure we use a builder that can leverage it (the default on linux will not)
	-$(DOCKER) buildx rm multiarch-multiplatform-builder
	$(DOCKER) buildx create --use --name=multiarch-multiplatform-builder
	$(DOCKER) run --rm --privileged multiarch/qemu-user-static --reset --credential yes --persistent yes
	# Register gcloud as a Docker credential helper.
	# Required for "docker buildx build --push".
	gcloud auth configure-docker --quiet
.PHONY: init-buildx

