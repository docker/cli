#!/usr/bin/env bash
set -eu -o pipefail

echo "Waiting for docker daemon to become available at $DOCKER_HOST"
while ! docker version > /dev/null; do
    sleep 0.3
done

docker version
