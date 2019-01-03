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

all: provisioner
.PHONY: all

test: 
	cd provisioner; make test
.PHONY: test
 
verify:
	./hack/verify-all.sh
.PHONY: verify

e2e:
	./hack/e2e.sh
.PHONY: e2e

provisioner:
	cd provisioner; make container
.PHONY: provisioner

push:
	cd provisioner; make push
.PHONY: push

clean:
.PHONY: clean
	cd provisioner; make clean
