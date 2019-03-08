FROM    koalaman/shellcheck-alpine:v0.6.0
RUN     apk add --no-cache bash make
WORKDIR /go/src/github.com/docker/cli
ENV     DISABLE_WARN_OUTSIDE_CONTAINER=1
COPY    . .
