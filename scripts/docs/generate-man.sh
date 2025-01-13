#!/usr/bin/env bash

set -eu

: "${MD2MAN_VERSION=v2.0.5}"

function clean() {
	rm -f go.mod
}

export GO111MODULE=auto
trap clean EXIT

./scripts/vendor init
# build gen-manpages
go build -mod=vendor -modfile=vendor.mod -tags manpages -o /tmp/gen-manpages ./man/generate.go
# build go-md2man
go build -mod=vendor -modfile=vendor.mod -o /tmp/go-md2man ./vendor/github.com/cpuguy83/go-md2man/v2

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
