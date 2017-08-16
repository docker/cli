FROM    debian:stretch-slim

RUN     apt-get update && \
        apt-get -y install make shellcheck && \
        apt-get clean

WORKDIR /go/src/github.com/docker/cli
ENV     DISABLE_WARN_OUTSIDE_CONTAINER=1
CMD     bash
