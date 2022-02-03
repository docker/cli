# syntax=docker/dockerfile:1.3

ARG BASE_VARIANT=alpine
ARG GO_VERSION=1.16.11
ARG XX_VERSION=1.0.0-rc.2
ARG GOVERSIONINFO_VERSION=v1.3.0
ARG GOTESTSUM_VERSION=v1.7.0
ARG BUILDX_VERSION=0.7.1

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-${BASE_VARIANT} AS gostable
FROM --platform=$BUILDPLATFORM golang:1.17rc1-${BASE_VARIANT} AS golatest

FROM gostable AS go-linux
FROM gostable AS go-darwin
FROM gostable AS go-windows-amd64
FROM gostable AS go-windows-386
FROM gostable AS go-windows-arm
FROM golatest AS go-windows-arm64
FROM go-windows-${TARGETARCH} AS go-windows

FROM --platform=$BUILDPLATFORM tonistiigi/xx:${XX_VERSION} AS xx

FROM go-${TARGETOS} AS build-base-alpine
COPY --from=xx / /
RUN apk add --no-cache bash clang lld llvm file git
WORKDIR /go/src/github.com/docker/cli

FROM build-base-alpine AS build-alpine
ARG TARGETPLATFORM
# gcc is installed for libgcc only
RUN xx-apk add --no-cache musl-dev gcc

FROM go-${TARGETOS} AS build-base-buster
COPY --from=xx / /
RUN apt-get update && apt-get install --no-install-recommends -y bash clang lld file
WORKDIR /go/src/github.com/docker/cli

FROM build-base-buster AS build-buster
ARG TARGETPLATFORM
RUN xx-apt install --no-install-recommends -y libc6-dev libgcc-8-dev

FROM build-base-${BASE_VARIANT} AS goversioninfo
ARG GOVERSIONINFO_VERSION
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOBIN=/out GO111MODULE=on go install "github.com/josephspurrier/goversioninfo/cmd/goversioninfo@${GOVERSIONINFO_VERSION}"

FROM build-base-${BASE_VARIANT} AS gotestsum
ARG GOTESTSUM_VERSION
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOBIN=/out GO111MODULE=on go install "gotest.tools/gotestsum@${GOTESTSUM_VERSION}" \
    && /out/gotestsum --version

FROM build-${BASE_VARIANT} AS build
# GO_LINKMODE defines if static or dynamic binary should be produced
ARG GO_LINKMODE=static
# GO_BUILDTAGS defines additional build tags
ARG GO_BUILDTAGS
# GO_STRIP strips debugging symbols if set
ARG GO_STRIP
# CGO_ENABLED manually sets if cgo is used
ARG CGO_ENABLED
# VERSION sets the version for the produced binary
ARG VERSION
# COMPANY_NAME sets the company that produced the windows binary
ARG COMPANY_NAME
COPY --from=goversioninfo /out/goversioninfo /usr/bin/goversioninfo
RUN --mount=type=bind,target=.,ro \
    --mount=type=cache,target=/root/.cache \
    --mount=from=dockercore/golang-cross:xx-sdk-extras,target=/xx-sdk,src=/xx-sdk \
    --mount=type=tmpfs,target=cli/winresources \
    # override the default behavior of go with xx-go
    xx-go --wrap && \
    # export GOCACHE=$(go env GOCACHE)/$(xx-info)$([ -f /etc/alpine-release ] && echo "alpine") && \
    TARGET=/out ./scripts/build/binary && \
    xx-verify $([ "$GO_LINKMODE" = "static" ] && echo "--static") /out/docker

FROM build-${BASE_VARIANT} AS test
COPY --from=gotestsum /out/gotestsum /usr/bin/gotestsum
ENV GO111MODULE=auto
RUN --mount=type=bind,target=.,rw \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
  gotestsum -- -coverprofile=/tmp/coverage.txt $(go list ./... | grep -vE '/vendor/|/e2e/')

FROM scratch AS test-coverage
COPY --from=test /tmp/coverage.txt /coverage.txt

FROM build-${BASE_VARIANT} AS build-plugins
ARG GO_LINKMODE=static
ARG GO_BUILDTAGS
ARG GO_STRIP
ARG CGO_ENABLED
ARG VERSION
RUN --mount=ro --mount=type=cache,target=/root/.cache \
    --mount=from=dockercore/golang-cross:xx-sdk-extras,target=/xx-sdk,src=/xx-sdk \
    xx-go --wrap && \
    TARGET=/out ./scripts/build/plugins e2e/cli-plugins/plugins/*

FROM build-base-alpine AS e2e-base-alpine
RUN apk add --no-cache build-base curl docker-compose openssl openssh-client

FROM build-base-buster AS e2e-base-buster
RUN apt-get update && apt-get install -y build-essential curl openssl openssh-client
ARG COMPOSE_VERSION=1.29.2
RUN curl -fsSL https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m) -o /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose

FROM docker/buildx-bin:${BUILDX_VERSION} AS buildx

FROM e2e-base-${BASE_VARIANT} AS e2e
ARG NOTARY_VERSION=v0.6.1
ADD --chmod=0755 https://github.com/theupdateframework/notary/releases/download/${NOTARY_VERSION}/notary-Linux-amd64 /usr/local/bin/notary
COPY e2e/testdata/notary/root-ca.cert /usr/share/ca-certificates/notary.cert
RUN echo 'notary.cert' >> /etc/ca-certificates.conf && update-ca-certificates
COPY --from=gotestsum /out/gotestsum /usr/bin/gotestsum
COPY --from=build /out ./build/
COPY --from=build-plugins /out ./build/
COPY --from=buildx /buildx /usr/libexec/docker/cli-plugins/docker-buildx
COPY . .
ENV DOCKER_BUILDKIT=1
ENV PATH=/go/src/github.com/docker/cli/build:$PATH
CMD ./scripts/test/e2e/entry

FROM build-base-${BASE_VARIANT} AS dev
COPY . .

FROM scratch AS binary
COPY --from=build /out .

FROM scratch AS plugins
COPY --from=build-plugins /out .
