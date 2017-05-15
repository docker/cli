#!/usr/bin/env bash
#
# Generate man pages for docker/docker
#

set -eu

mkdir -p ./man/man1

MD2MAN_REPO=github.com/cpuguy83/go-md2man
MD2MAN_COMMIT=a65d4d2de4d5f7c74868dfa9b202a3c8be315aaa

( 
	go get -d "$MD2MAN_REPO"
	cd "$GOPATH"/src/"$MD2MAN_REPO"
	git checkout "$MD2MAN_COMMIT" &> /dev/null
	go install "$MD2MAN_REPO"
)

VENDOR_MD5="$(md5sum vendor.conf)"
cp vendor.conf /tmp/vendor.conf
cp man/vendor.tmp vendor.conf

grep -v '^#' man/vendor.tmp | while read dep; do
	vndr $(echo "$dep" | cut -d' ' -f1)
done 

cp /tmp/vendor.conf vendor.conf
[ "$(md5sum vendor.conf)" != "$VENDOR_MD5" ] && echo "/tmp/vendor.conf unexpectedly changed. Expected $VENDOR_MD5" && exit 1

# Generate man pages from cobra commands
go build -o /tmp/gen-manpages ./man
/tmp/gen-manpages --root . --target ./man/man1

# cleanup
grep -v '^#' man/vendor.tmp | while read dep; do
	rm -rf vendor/$(echo "$dep" | cut -d' ' -f1)
done
git checkout vendor/

# Generate legacy pages from markdown
./man/md2man-all.sh -q
