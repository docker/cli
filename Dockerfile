# syntax=docker/dockerfile:1

ARG BASE_VARIANT=alpine
ARG GO_VERSION=1.20.13
ARG ALPINE_VERSION=3.18
ARG XX_VERSION=1.2.1
ARG GOVERSIONINFO_VERSION=v1.3.0
ARG GOTESTSUM_VERSION=v1.10.0
ARG BUILDX_VERSION=0.11.2
ARG COMPOSE_VERSION=v2.22.0

FROM --platform=$BUILDPLATFORM tonistiigi/xx:${XX_VERSION} AS xx

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build-base-alpine
ENV GOTOOLCHAIN=local
COPY --link --from=xx / /
RUN apk add --no-cache bash clang lld llvm file git
WORKDIR /go/src/github.com/docker/cli

FROM build-base-alpine AS build-alpine
ARG TARGETPLATFORM
# gcc is installed for libgcc only
RUN xx-apk add --no-cache musl-dev gcc

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bullseye AS build-base-bullseye
ENV GOTOOLCHAIN=local
COPY --link --from=xx / /
RUN apt-get update && apt-get install --no-install-recommends -y bash clang lld llvm file
WORKDIR /go/src/github.com/docker/cli

FROM build-base-bullseye AS build-bullseye
ARG TARGETPLATFORM
RUN xx-apt-get install --no-install-recommends -y libc6-dev libgcc-10-dev
# workaround for issue with llvm 11 for darwin/amd64 platform:
#  # github.com/docker/cli/cmd/docker
#  /usr/local/go/pkg/tool/linux_amd64/link: /usr/local/go/pkg/tool/linux_amd64/link: running strip failed: exit status 1
#  llvm-strip: error: unsupported load command (cmd=0x5)
# more info: https://github.com/docker/cli/pull/3717
# FIXME: remove once llvm 12 available on debian
RUN [ "$TARGETPLATFORM" != "darwin/amd64" ] || ln -sfnT /bin/true /usr/bin/llvm-strip

FROM build-base-${BASE_VARIANT} AS goversioninfo
ARG GOVERSIONINFO_VERSION
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOBIN=/out GO111MODULE=on CGO_ENABLED=0 go install "github.com/josephspurrier/goversioninfo/cmd/goversioninfo@${GOVERSIONINFO_VERSION}"

FROM build-base-${BASE_VARIANT} AS gotestsum
ARG GOTESTSUM_VERSION
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOBIN=/out GO111MODULE=on CGO_ENABLED=0 go install "gotest.tools/gotestsum@${GOTESTSUM_VERSION}" \
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
# PACKAGER_NAME sets the company that produced the windows binary
ARG PACKAGER_NAME
COPY --link --from=goversioninfo /out/goversioninfo /usr/bin/goversioninfo
# in bullseye arm64 target does not link with lld so configure it to use ld instead
RUN [ ! -f /etc/alpine-release ] && xx-info is-cross && [ "$(xx-info arch)" = "arm64" ] && XX_CC_PREFER_LINKER=ld xx-clang --setup-target-triple || true
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
COPY --link --from=gotestsum /out/gotestsum /usr/bin/gotestsum
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
RUN apk add --no-cache build-base curl openssl openssh-client

FROM build-base-bullseye AS e2e-base-bullseye
RUN apt-get update && apt-get install -y build-essential curl openssl openssh-client

FROM docker/buildx-bin:${BUILDX_VERSION}   AS buildx
FROM docker/compose-bin:${COMPOSE_VERSION} AS compose

FROM e2e-base-${BASE_VARIANT} AS e2e
ARG NOTARY_VERSION=v0.6.1
ADD --chmod=0755 https://github.com/theupdateframework/notary/releases/download/${NOTARY_VERSION}/notary-Linux-amd64 /usr/local/bin/notary
COPY --link e2e/testdata/notary/root-ca.cert /usr/share/ca-certificates/notary.cert
RUN echo 'notary.cert' >> /etc/ca-certificates.conf && update-ca-certificates
COPY --link --from=gotestsum /out/gotestsum /usr/bin/gotestsum
COPY --link --from=build /out ./build/
COPY --link --from=build-plugins /out ./build/
COPY --link --from=buildx  /buildx         /usr/libexec/docker/cli-plugins/docker-buildx
COPY --link --from=compose /docker-compose /usr/libexec/docker/cli-plugins/docker-compose
COPY --link . .
ENV DOCKER_BUILDKIT=1
ENV PATH=/go/src/github.com/docker/cli/build:$PATH
CMD ./scripts/test/e2e/entry

FROM build-base-${BASE_VARIANT} AS dev
COPY --link . .

FROM scratch AS plugins
COPY --from=build-plugins /out .

FROM scratch AS binary
COPY --from=build /out .
