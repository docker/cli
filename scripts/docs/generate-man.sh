#!/usr/bin/env bash

set -eu

: "${MD2MAN_VERSION=v2.0.5}"

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
  ./scripts/vendor init
  # install go-md2man and copy man/tools.go in root folder
  # to be able to fetch the required dependencies
  go mod edit -modfile=vendor.mod -require=github.com/cpuguy83/go-md2man/v2@${MD2MAN_VERSION}
  cp man/tools.go .
  # update vendor
  ./scripts/vendor update
  # build gen-manpages
  go build -mod=vendor -modfile=vendor.mod -tags manpages -o /tmp/gen-manpages ./man/generate.go
  # build go-md2man
  go build -mod=vendor -modfile=vendor.mod -o /tmp/go-md2man ./vendor/github.com/cpuguy83/go-md2man/v2
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
