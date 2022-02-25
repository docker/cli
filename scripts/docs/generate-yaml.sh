#!/usr/bin/env bash

set -eu

: "${CLI_DOCS_TOOL_VERSION=v0.3.1}"

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
  # install cli-docs-tool and copy docs/tools.go in root folder
  # to be able to fetch the required depedencies
  go mod edit -modfile=vendor.mod -require=github.com/docker/cli-docs-tool@${CLI_DOCS_TOOL_VERSION}
  cp docs/tools.go .
  # update vendor
  ./scripts/vendor update
  # build docsgen
  go build -mod=vendor -modfile=vendor.mod -tags docsgen -o /tmp/docsgen ./docs/generate.go
)

mkdir -p docs/yaml
set -x
/tmp/docsgen --source "$(pwd)/docs/reference/commandline" --target "$(pwd)/docs/yaml"
