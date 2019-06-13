ARG GO_VERSION=1.12.6

FROM docker/containerd-shim-process:a4d1531 AS containerd-shim-process

# Use Debian based image as docker-compose requires glibc.
FROM golang:${GO_VERSION}

RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    openssl \
    btrfs-tools \
    libapparmor-dev \
    libseccomp-dev \
    iptables \
    openssh-client \
    && rm -rf /var/lib/apt/lists/*

ARG COMPOSE_VERSION=1.21.2
RUN curl -L https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-`uname -s`-`uname -m` -o /usr/local/bin/docker-compose \
    && chmod +x /usr/local/bin/docker-compose

ARG NOTARY_VERSION=v0.6.1
RUN curl -Ls https://github.com/theupdateframework/notary/releases/download/${NOTARY_VERSION}/notary-Linux-amd64 -o /usr/local/bin/notary \
    && chmod +x /usr/local/bin/notary

ARG GOTESTSUM_VERSION=0.3.4
RUN curl -Ls https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_linux_amd64.tar.gz -o gotestsum.tar.gz \
    && tar -xf gotestsum.tar.gz gotestsum \
    && mv gotestsum /usr/local/bin/gotestsum \
    && rm gotestsum.tar.gz 

ENV CGO_ENABLED=0 \
    DISABLE_WARN_OUTSIDE_CONTAINER=1 \
    PATH=/go/src/github.com/docker/cli/build:$PATH
WORKDIR /go/src/github.com/docker/cli

# Trust notary CA cert.
COPY e2e/testdata/notary/root-ca.cert /usr/share/ca-certificates/notary.cert
RUN echo 'notary.cert' >> /etc/ca-certificates.conf && update-ca-certificates

COPY . .
ARG VERSION
ARG GITCOMMIT
ENV VERSION=${VERSION} GITCOMMIT=${GITCOMMIT}
RUN ./scripts/build/binary
RUN ./scripts/build/plugins e2e/cli-plugins/plugins/*

CMD ./scripts/test/e2e/entry
