FROM golang:1.17.13-alpine3.16 as builder-base
RUN apk add make bash git openssh build-base curl

#
# STAGE - Build stage, calls make with given target argument (defaults to all make target)
#
FROM builder-base as builder
ARG target=all
ENV RUN_LOCAL=1
RUN mkdir -p /go/src
ADD . /go/src/
WORKDIR /go/src
RUN make $target

#
# STAGE - Test Build stage, calls make with given target argument (defaults to all make target). Valid for testing purposes only as tests require a specific (non-root) user access for directories read/write access.
#
FROM builder-base as test-builder
ARG target=all
ENV GROUP=test-group
ENV USER=test-user
ENV UID=12345
ENV GID=23456
ENV RUN_LOCAL=1
RUN addgroup -S $GROUP
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "$(pwd)" \
    --ingroup "$GROUP" \
    --no-create-home \
    --uid "$UID" \
    "$USER"
USER $USER
RUN mkdir -p /go/src
ADD . /go/src/
WORKDIR /go/src
RUN make $target
