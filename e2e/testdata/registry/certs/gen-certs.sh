#!/bin/sh
set -eu

# Regenerate test certificates for the TLS-enabled private registry.
# Run this from the repository root or from e2e/testdata/registry/certs/.

cd "$(dirname "$0")"

# --- CA ---
openssl genrsa -out ca.key 2048
openssl req -new -x509 -days 3650 \
	-key ca.key \
	-subj '/CN=Test CA (TLS Registry)' \
	-out ca.crt

# --- Server cert for tlsregistry (signed by CA) ---
cat > openssl-tlsregistry.cnf <<-EOF
	[v3_req]
	subjectAltName=DNS:tlsregistry
EOF
openssl genrsa -out tlsregistry.key 2048
openssl req -new \
	-key tlsregistry.key \
	-subj '/CN=tlsregistry' \
	-out tlsregistry.csr
openssl x509 -req -days 3650 \
	-in tlsregistry.csr \
	-CA ca.crt -CAkey ca.key \
	-CAcreateserial \
	-out tlsregistry.crt \
	-extfile openssl-tlsregistry.cnf \
	-extensions v3_req
rm -f tlsregistry.csr ca.srl openssl-tlsregistry.cnf
