#!/usr/bin/env bash

set -Eeuo pipefail

export GO111MODULE=auto

# temporary "go.mod" to make -modfile= work
touch go.mod

function clean() {
  rm -f "$(pwd)/go.mod"
}

trap clean EXIT

# build docsgen
go build -mod=vendor -modfile=vendor.mod -tags docsgen -o /tmp/docsgen ./docs/generate/generate.go

mkdir -p docs/yaml
set -x
/tmp/docsgen --formats yaml --source "$(pwd)/docs/reference/commandline" --target "$(pwd)/docs/yaml"
