#!/usr/bin/env bash

set -eu

: "${CLI_DOCS_TOOL_VERSION=v0.6.0}"

export GO111MODULE=auto

function clean {
  rm -rf "$buildir"
  if [ -f "$(pwd)/docs/reference/commandline/docker.md" ]; then
    mv "$(pwd)/docs/reference/commandline/docker.md" "$(pwd)/docs/reference/commandline/cli.md"
  fi
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
  cp docs/generate/tools.go .
  # update vendor
  ./scripts/vendor update
  # build docsgen
  go build -mod=vendor -modfile=vendor.mod -tags docsgen -o /tmp/docsgen ./docs/generate/generate.go
)

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
