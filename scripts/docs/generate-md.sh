#!/usr/bin/env bash

set -eu

(
  set -x
  go run -mod=vendor -modfile=vendor.mod -tags docsgen ./docs/generate/generate.go --formats md --source "./docs/reference/commandline" --target "./docs/reference/commandline"
)

# remove generated help.md file
rm "$(pwd)/docs/reference/commandline/help.md" >/dev/null 2>&1 || true
