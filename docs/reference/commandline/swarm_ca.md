---
title: "swarm ca"
description: "The swarm ca command description and usage"
keywords: "swarm, ca"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# swarm ca

```markdown
Usage:	docker swarm ca [OPTIONS]

Manage root CA

Options:
      --ca-cert pem-file          Path to the PEM-formatted root CA certificate to use for the new cluster
      --ca-key pem-file           Path to the PEM-formatted root CA key to use for the new cluster
      --cert-expiry duration      Validity period for node certificates (ns|us|ms|s|m|h) (default 2160h0m0s)
  -d, --detach                    Exit immediately instead of waiting for the root rotation to converge
      --external-ca external-ca   Specifications of one or more certificate signing endpoints in the form: "protocol=<protocol>,url=<url>"
      --help                      Print usage
  -q, --quiet                     Suppress progress output
      --rotate                    Rotate the swarm CA - if no certificate or key are provided, new ones will be generated
```

## Description

View or rotate the current swarm CA certificate. This command must target a manager node.  If no CA certificate
is provided, a new certificate and key will be generated and used.

A CA certificate, along with either a CA key and/or an external CA URL, may optionally be provided instead.

## Examples

Run the `docker swarm ca` command without any options to view the current root CA certificate
in PEM format.

```bash
$ docker swarm ca
-----BEGIN CERTIFICATE-----
MIIBazCCARCgAwIBAgIUJPzo67QC7g8Ebg2ansjkZ8CbmaswCgYIKoZIzj0EAwIw
EzERMA8GA1UEAxMIc3dhcm0tY2EwHhcNMTcwNTAzMTcxMDAwWhcNMzcwNDI4MTcx
MDAwWjATMREwDwYDVQQDEwhzd2FybS1jYTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABKL6/C0sihYEb935wVPRA8MqzPLn3jzou0OJRXHsCLcVExigrMdgmLCC+Va4
+sJ+SLVO1eQbvLHH8uuDdF/QOU6jQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMB
Af8EBTADAQH/MB0GA1UdDgQWBBSfUy5bjUnBAx/B0GkOBKp91XvxzjAKBggqhkjO
PQQDAgNJADBGAiEAnbvh0puOS5R/qvy1PMHY1iksYKh2acsGLtL/jAIvO4ACIQCi
lIwQqLkJ48SQqCjG1DBTSBsHmMSRT+6mE2My+Z3GKA==
-----END CERTIFICATE-----
```

Pass the `--rotate` flag (and optionally a `--ca-cert`, along with a `--ca-key` or
`--external-ca` parameter flag), in order to rotate the current swarm root CA.

```
$ docker swarm ca --rotate
desired root digest: sha256:05da740cf2577a25224c53019e2cce99bcc5ba09664ad6bb2a9425d9ebd1b53e
  rotated TLS certificates:  [=========================>                         ] 1/2 nodes
  rotated CA certificates:   [>                                                  ] 0/2 nodes
```

Once the rotation os finished (all the progress bars have completed) the now-current
CA certificate will be printed:

```
$ docker swarm ca --rotate
desired root digest: sha256:05da740cf2577a25224c53019e2cce99bcc5ba09664ad6bb2a9425d9ebd1b53e
  rotated TLS certificates:  [==================================================>] 2/2 nodes
  rotated CA certificates:   [==================================================>] 2/2 nodes
-----BEGIN CERTIFICATE-----
MIIBazCCARCgAwIBAgIUFynG04h5Rrl4lKyA4/E65tYKg8IwCgYIKoZIzj0EAwIw
EzERMA8GA1UEAxMIc3dhcm0tY2EwHhcNMTcwNTE2MDAxMDAwWhcNMzcwNTExMDAx
MDAwWjATMREwDwYDVQQDEwhzd2FybS1jYTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABC2DuNrIETP7C7lfiEPk39tWaaU0I2RumUP4fX4+3m+87j0DU0CsemUaaOG6
+PxHhGu2VXQ4c9pctPHgf7vWeVajQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMB
Af8EBTADAQH/MB0GA1UdDgQWBBSEL02z6mCI3SmMDmITMr12qCRY2jAKBggqhkjO
PQQDAgNJADBGAiEA263Eb52+825EeNQZM0AME+aoH1319Zp9/J5ijILW+6ACIQCg
gyg5u9Iliel99l7SuMhNeLkrU7fXs+Of1nTyyM73ig==
-----END CERTIFICATE-----
```

### `--rotate`

Root CA Rotation is recommended if one or more of the swarm managers have been
compromised, so that those managers can no longer connect to or be trusted by
any other node in the cluster.

Alternately, root CA rotation can be used to give control of the swarm CA
to an external CA, or to take control back from an external CA.

The `--rotate` flag does not require any parameters to do a rotation, but you can
optionally specify a certificate and key, or a certificate and external CA URL,
and those will be used instead of an automatically-generated certificate/key pair.

Because the root CA key should be kept secret, if provided it will not be visible
when viewing swarm any information via the CLI or API.

The root CA rotation will not be completed until all registered nodes have
rotated their TLS certificates.  If the rotation is not completing within a
reasonable amount of time, try running
`docker node ls --format '{{.ID}} {{.Hostname}} {{.Status}} {{.TLSStatus}}'` to
see if any nodes are down or otherwise unable to rotate TLS certificates.


### `--detach`

Initiate the root CA rotation, but do not wait for the completion of or display the
progress of the rotation.

## `--external-ca`

Provide an external CA URL to the managers - this CA will be used to issue all new
node certificates.  One or more of these flags can be passed in order to provide
multiple alternative URLs - the swarm will try each one in order before falling
back on the next. If a key and an external CA are both provided, swarm will only
use the external CA.

This flag must be passed in conjunction with the `--ca-cert` flag, to specify
which CA certificate will be used by these external URLs to sign certificates.

The only protocol currently supported for external CAs is the CFSSL signing
protocol, and the URL provided should be to the [`/api/v1/cfssl/sign`](https://github.com/cloudflare/cfssl/blob/master/doc/api/endpoint_sign.txt)
endpoint on a CFSSL server.

The URL must be HTTPS, and the TLS certificate for the external URL must be signed
by the CA certificate provided via the `--ca-cert` flag.

Example usage:

`--external-ca=protocol=cfssl,url=https://my.cfssl.server/api/v1/cfssl/sign`

External CAs can be removed though by passing an empty string, e.g.
`--external-ca=""`.


## Related commands

* [swarm init](swarm_init.md)
* [swarm join](swarm_join.md)
* [swarm join-token](swarm_join_token.md)
* [swarm leave](swarm_leave.md)
* [swarm unlock](swarm_unlock.md)
* [swarm unlock-key](swarm_unlock_key.md)
