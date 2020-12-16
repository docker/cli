#!/usr/bin/env bash
# Generate man pages for docker/cli
set -eu -o pipefail

mkdir -p ./man/man1

if ! command -v go-md2man &> /dev/null; then
	# yay, go install creates a binary named "v2" ¯\_(ツ)_/¯
	go build -o "/go/bin/go-md2man" ./vendor/github.com/cpuguy83/go-md2man/v2
fi

# Generate man pages from cobra commands
go build -o /tmp/gen-manpages github.com/docker/cli/man
/tmp/gen-manpages --root "$(pwd)" --target "$(pwd)/man/man1"

# Generate legacy pages from markdown
./man/md2man-all.sh -q
