#!/usr/bin/env bash

set -Eeuo pipefail

export GO111MODULE=auto

# temporary "go.mod" to make -modfile= work
touch go.mod

function clean {
  rm -f "$(pwd)/go.mod"
}

trap clean EXIT

# build docsgen
go build -mod=vendor -modfile=vendor.mod -tags docsgen -o /tmp/docsgen ./docs/generate/generate.go

(
  set -x
  /tmp/docsgen --formats md --source "$(pwd)/docs/reference/commandline" --target "$(pwd)/docs/reference/commandline"
)

# remove generated help.md file
rm "$(pwd)/docs/reference/commandline/help.md" >/dev/null 2>&1 || true
