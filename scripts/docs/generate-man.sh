#!/usr/bin/env bash

set -eu

: "${MD2MAN_VERSION=v2.0.6}"

export GO111MODULE=auto

function clean {
  rm -rf "$buildir"
}

buildir=$(mktemp -d -t docker-cli-docsgen.XXXXXXXXXX)
trap clean EXIT

(
  set -x
  cp -r . "$buildir/"
  cd "$buildir"
  # init dummy go.mod
  # install go-md2man and copy man/tools.go in root folder
  # to be able to fetch the required dependencies
  go get github.com/cpuguy83/go-md2man/v2@${MD2MAN_VERSION}
  cp man/tools.go .
  # build gen-manpages
  go build -tags manpages -o /tmp/gen-manpages ./man/generate.go
  # build go-md2man
  GOBIN=$buildir/ go install github.com/cpuguy83/go-md2man/v2@${MD2MAN_VERSION}
)

mkdir -p man/man1
(set -x ; /tmp/gen-manpages --root "." --target "$(pwd)/man/man1")

(
  cd man
  for FILE in *.md; do
    base="$(basename "$FILE")"
    name="${base%.md}"
    num="${name##*.}"
    if [ -z "$num" ] || [ "$name" = "$num" ]; then
      # skip files that aren't of the format xxxx.N.md (like README.md)
      continue
    fi
    mkdir -p "./man${num}"
    (set -x ; /tmp/go-md2man -in "$FILE" -out "./man${num}/${name}")
  done
)
