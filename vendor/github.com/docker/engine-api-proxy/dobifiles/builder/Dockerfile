
FROM    golang:1.7.5-alpine3.5

RUN     apk add -U git bash curl tree
RUN     git config --global http.https://gopkg.in.followRedirects true

RUN     export GLIDE=v0.12.3; \
        export SRC=https://github.com/Masterminds/glide/releases/download/; \
        curl -sL ${SRC}/${GLIDE}/glide-${GLIDE}-linux-amd64.tar.gz | \
        tar -xz linux-amd64/glide && \
        mv linux-amd64/glide /usr/bin/glide && \
        chmod +x /usr/bin/glide

RUN     go get github.com/dnephin/filewatcher && \
        cp /go/bin/filewatcher /usr/bin/ && \
        rm -rf /go/src/* /go/pkg/* /go/bin/*

WORKDIR /go/src/github.com/docker/engine-api-proxy
RUN     addgroup docker # used for pinata tests
ENV     PS1="# "
ENV     CGO_ENABLED=0
