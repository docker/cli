#!/usr/bin/env sh
set -eu

# This script is used to generate the test-certificates in the notary-server and
# evil-notary-server directories. Run this script to update the certificates if
# they expire.
GO111MODULE=off go get -u github.com/dmcgowan/quicktls
cd notary
quicktls -org=Docker -with-san notary-server notaryserver evil-notary-server evilnotaryserver localhost 127.0.0.1
cat ca.pem >> notary-server.cert
mv ca.pem root-ca.cert
cp notary-server.cert notary-server.key root-ca.cert ../notary-evil/
