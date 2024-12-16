#!/usr/bin/env bash

set -eu

: "${CLI_DOCS_TOOL_VERSION=v0.8.0}"

function clean() {
	rm -f go.mod
}

export GO111MODULE=auto
trap clean EXIT

./scripts/vendor init
# build docsgen
go build -mod=vendor -modfile=vendor.mod -tags docsgen -o /tmp/docsgen ./docs/generate/generate.go

(
  set -x
  /tmp/docsgen --formats md --source "$(pwd)/docs/reference/commandline" --target "$(pwd)/docs/reference/commandline"
)

# remove generated help.md file
rm "$(pwd)/docs/reference/commandline/help.md" >/dev/null 2>&1 || true
