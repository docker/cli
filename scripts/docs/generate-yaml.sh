#!/bin/sh

go build -o build/yaml-docs-generator github.com/docker/cli/docs/yaml
mkdir docs/yaml/gen
build/yaml-docs-generator --root $(pwd) --target $(pwd)/docs/yaml/gen
