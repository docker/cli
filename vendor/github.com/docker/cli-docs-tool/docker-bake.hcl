// Copyright 2021 cli-docs-tool authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

target "_common" {
  args = {
    BUILDKIT_CONTEXT_KEEP_GIT_DIR = 1
  }
}

group "default" {
  targets = ["test"]
}

group "validate" {
  targets = ["lint", "vendor-validate", "license-validate"]
}

target "lint" {
  inherits = ["_common"]
  target = "lint"
  output = ["type=cacheonly"]
}

target "vendor-validate" {
  inherits = ["_common"]
  target = "vendor-validate"
  output = ["type=cacheonly"]
}

target "vendor-update" {
  inherits = ["_common"]
  target = "vendor-update"
  output = ["."]
}

target "test" {
  inherits = ["_common"]
  target = "test-coverage"
  output = ["."]
}

target "license-validate" {
  inherits = ["_common"]
  target = "license-validate"
  output = ["type=cacheonly"]
}

target "license-update" {
  inherits = ["_common"]
  target = "license-update"
  output = ["."]
}
