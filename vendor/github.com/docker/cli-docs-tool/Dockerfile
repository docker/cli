# syntax=docker/dockerfile:1

# Copyright 2021 cli-docs-tool authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG GO_VERSION="1.24"
ARG XX_VERSION="1.6.1"
ARG GOLANGCI_LINT_VERSION="v2.1.5"
ARG ADDLICENSE_VERSION="v1.1.1"

ARG LICENSE_ARGS="-c cli-docs-tool -l apache"
ARG LICENSE_FILES=".*\(Dockerfile\|\.go\|\.hcl\|\.sh\)"

FROM golangci/golangci-lint:${GOLANGCI_LINT_VERSION}-alpine AS golangci-lint
FROM --platform=$BUILDPLATFORM tonistiigi/xx:${XX_VERSION} AS xx

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS base
RUN apk add --no-cache cpio findutils git linux-headers
ENV CGO_ENABLED=0
WORKDIR /src
COPY --link --from=xx / /

FROM base AS addlicense
ARG ADDLICENSE_VERSION
ARG TARGETPLATFORM
RUN --mount=target=/root/.cache,type=cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
  set -ex
  xx-go install "github.com/google/addlicense@${ADDLICENSE_VERSION}"
  mkdir /out
  if ! xx-info is-cross; then
    mv /go/bin/addlicense /out
  else
    mv /go/bin/*/addlicense* /out
  fi
EOT

FROM base AS vendored
RUN --mount=type=bind,target=.,rw \
  --mount=type=cache,target=/go/pkg/mod \
  go mod tidy && go mod download && \
  mkdir /out && cp go.mod go.sum /out

FROM scratch AS vendor-update
COPY --from=vendored /out /

FROM vendored AS vendor-validate
RUN --mount=type=bind,target=.,rw <<EOT
  set -e
  git add -A
  cp -rf /out/* .
  diff=$(git status --porcelain -- go.mod go.sum)
  if [ -n "$diff" ]; then
    echo >&2 'ERROR: Vendor result differs. Please vendor your package with "docker buildx bake vendor"'
    echo "$diff"
    exit 1
  fi
EOT

FROM base AS lint
RUN --mount=type=bind,target=. \
  --mount=type=cache,target=/root/.cache \
  --mount=from=golangci-lint,source=/usr/bin/golangci-lint,target=/usr/bin/golangci-lint \
  golangci-lint run ./...

FROM base AS license-set
ARG LICENSE_ARGS
ARG LICENSE_FILES
RUN --mount=type=bind,target=.,rw \
  --mount=from=addlicense,source=/out/addlicense,target=/usr/bin/addlicense \
  find . -regex "${LICENSE_FILES}" | xargs addlicense ${LICENSE_ARGS} \
  && mkdir /out \
  && find . -regex "${LICENSE_FILES}" | cpio -pdm /out

FROM scratch AS license-update
COPY --from=set /out /

FROM base AS license-validate
ARG LICENSE_ARGS
ARG LICENSE_FILES
RUN --mount=type=bind,target=. \
  --mount=from=addlicense,source=/out/addlicense,target=/usr/bin/addlicense \
  find . -regex "${LICENSE_FILES}" | xargs addlicense -check ${LICENSE_ARGS}

FROM vendored AS test
RUN --mount=type=bind,target=. \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
  go test -v -coverprofile=/tmp/coverage.txt -covermode=atomic ./...

FROM scratch AS test-coverage
COPY --from=test /tmp/coverage.txt /coverage.txt
