#!/usr/bin/env bash
#
# Build plugins examples for the host OS/ARCH
#

set -eu -o pipefail

source ./scripts/build/.variables

for p in cli-plugins/examples/* "$@" ; do
    [ -d "$p" ] || continue

    n=$(basename "$p")
    TARGET_PLUGIN="$(dirname "${TARGET}")/plugins-${GOOS}-${GOARCH}/docker-${n}"
    mkdir -p "$(dirname "${TARGET_PLUGIN}")"

    echo "Building $GO_LINKMODE $(basename "${TARGET_PLUGIN}")"
    (set -x ; CGO_ENABLED=0 GO111MODULE=auto go build -o "${TARGET_PLUGIN}" -tags "${GO_BUILDTAGS}" -ldflags "${LDFLAGS}" ${GO_BUILDMODE} "github.com/docker/cli/${p}")
done
