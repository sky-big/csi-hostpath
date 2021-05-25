# Copyright 2019 The Kubernetes Authors.
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

.PHONY: build
build:
	./hack/build.sh driver hostpath.csi.kubernetes.io

# image
.PHONY: build-image
build-image: build
	./hack/build-image.sh

.PHONY: push-image
push-image: build
	./hack/push-image.sh

.PHONY: clean
clean:
	./hack/clean.sh