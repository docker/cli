#!/usr/bin/env bash

set -Eeuo pipefail

export GO111MODULE=auto

# temporary "go.mod" to make -modfile= work
touch go.mod

function clean {
  rm -f "$(pwd)/go.mod"
  if [ -f "$(pwd)/docs/reference/commandline/docker.md" ]; then
    mv "$(pwd)/docs/reference/commandline/docker.md" "$(pwd)/docs/reference/commandline/cli.md"
  fi
}

trap clean EXIT

# build docsgen
go build -mod=vendor -modfile=vendor.mod -tags docsgen -o /tmp/docsgen ./docs/generate/generate.go

# yaml generation on docs repo needs the cli.md file: https://github.com/docker/cli/pull/3924#discussion_r1059986605
# but markdown generation docker.md atm. While waiting for a fix in cli-docs-tool
# we need to first move the cli.md file to docker.md, do the generation and
# then move it back in trap handler.
mv "$(pwd)/docs/reference/commandline/cli.md" "$(pwd)/docs/reference/commandline/docker.md"

(
  set -x
  /tmp/docsgen --formats md --source "$(pwd)/docs/reference/commandline" --target "$(pwd)/docs/reference/commandline"
)

# remove generated help.md file
rm "$(pwd)/docs/reference/commandline/help.md" >/dev/null 2>&1 || true
