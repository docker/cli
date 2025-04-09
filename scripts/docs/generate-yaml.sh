#!/usr/bin/env bash

set -eu

mkdir -p docs/yaml
set -x
go run -mod=vendor -modfile=vendor.mod -tags docsgen ./docs/generate/generate.go --formats yaml --source "./docs/reference/commandline" --target "./docs/yaml"
