#!/usr/bin/env bash
#
# Generate man pages for docker/docker
#

set -eu

mkdir -p ./man/man1

MD2MAN_REPO=github.com/cpuguy83/go-md2man
MD2MAN_COMMIT=$(grep -F "$MD2MAN_REPO" vendor.conf | cut -d' ' -f2)

(
	go get -d "$MD2MAN_REPO"
	cd "$GOPATH"/src/"$MD2MAN_REPO"
	git checkout "$MD2MAN_COMMIT" &> /dev/null
	go install "$MD2MAN_REPO"
)

# Generate man pages from cobra commands
go build -o /tmp/gen-manpages github.com/docker/cli/man
/tmp/gen-manpages --root . --target ./man/man1

# Generate legacy pages from markdown
./man/md2man-all.sh -q
