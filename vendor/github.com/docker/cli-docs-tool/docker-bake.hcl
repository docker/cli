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

group "default" {
  targets = ["test"]
}

group "validate" {
  targets = ["lint", "vendor-validate", "license-validate"]
}

target "lint" {
  target = "lint"
  output = ["type=cacheonly"]
}

target "vendor-validate" {
  target = "vendor-validate"
  output = ["type=cacheonly"]
}

target "vendor-update" {
  target = "vendor-update"
  output = ["."]
}

target "test" {
  target = "test-coverage"
  output = ["."]
}

target "license-validate" {
  target = "license-validate"
  output = ["type=cacheonly"]
}

target "license-update" {
  target = "license-update"
  output = ["."]
}
