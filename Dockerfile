# syntax=docker/dockerfile:1.3

ARG BASE_VARIANT=alpine
ARG GO_VERSION=1.16.8
ARG XX_VERSION=1.0.0-rc.2

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
RUN apk add --no-cache clang lld llvm file git
WORKDIR /src

FROM build-base-alpine AS build-alpine
ARG TARGETPLATFORM
# gcc is installed for libgcc only
RUN xx-apk add --no-cache musl-dev gcc

FROM go-${TARGETOS} AS build-base-buster
COPY --from=xx / /
RUN apt-get update && apt-get install --no-install-recommends -y clang lld file
WORKDIR /src

FROM build-base-buster AS build-buster
ARG TARGETPLATFORM
RUN xx-apt install --no-install-recommends -y libc6-dev libgcc-8-dev

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
RUN --mount=ro --mount=type=cache,target=/root/.cache \
    --mount=from=dockercore/golang-cross:xx-sdk-extras,target=/xx-sdk,src=/xx-sdk \
    --mount=type=tmpfs,target=cli/winresources \
    xx-go --wrap && \
    # export GOCACHE=$(go env GOCACHE)/$(xx-info)$([ -f /etc/alpine-release ] && echo "alpine") && \
    TARGET=/out ./scripts/build/binary && \
    xx-verify $([ "$GO_LINKMODE" = "static" ] && echo "--static") /out/docker

FROM build-base-${BASE_VARIANT} AS dev 
COPY . .

FROM scratch AS binary
COPY --from=build /out .
