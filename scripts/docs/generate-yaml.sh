#!/usr/bin/env bash

set -eu

: "${CLI_DOCS_TOOL_VERSION=v0.9.0}"

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
  # install cli-docs-tool and copy docs/tools.go in root folder
  # to be able to fetch the required dependencies
  go install github.com/docker/cli-docs-tool@${CLI_DOCS_TOOL_VERSION}
  cp docs/generate/tools.go .
  # build docsgen
  go build -tags docsgen -o /tmp/docsgen ./docs/generate/generate.go
)

mkdir -p docs/yaml
set -x
/tmp/docsgen --formats yaml --source "$(pwd)/docs/reference/commandline" --target "$(pwd)/docs/yaml"
