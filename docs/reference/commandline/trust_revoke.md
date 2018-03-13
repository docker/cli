---
title: "trust revoke"
description: "The revoke command description and usage"
keywords: "revoke, notary, trust"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# trust revoke

```markdown
Usage:  docker trust revoke [OPTIONS] IMAGE[:TAG]

Remove trust for an image

Options:
      --help   Print usage
  -y, --yes    Do not prompt for confirmation
```

## Description

`docker trust revoke` removes signatures from tags in signed repositories.

## Examples

### Revoke signatures from a signed tag

Here's an example of a repo with two signed tags:


```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                              SIGNERS
red                 852cc04935f930a857b630edc4ed6131e91b22073bcc216698842e44f64d2943    alice
blue                f1c38dbaeeb473c36716f6494d803fbfbe9d8a76916f7c0093f227821e378197    alice, bob

List of signers and their keys for example/trust-demo:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	ecc457614c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

When `alice`, one of the signers, runs `docker trust revoke`:

```bash
$ docker trust revoke example/trust-demo:red
Enter passphrase for delegation key with ID 27d42a8:
Successfully deleted signature for example/trust-demo:red
```

After revocation, the tag is removed from the list of released tags:

```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                              SIGNERS
blue                f1c38dbaeeb473c36716f6494d803fbfbe9d8a76916f7c0093f227821e378197    alice, bob

List of signers and their keys for example/trust-demo:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	ecc457614c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

### Revoke signatures on all tags in a repository

When no tag is specified, `docker trust` revokes all signatures that you have a signing key for.

```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                              SIGNERS
red                 852cc04935f930a857b630edc4ed6131e91b22073bcc216698842e44f64d2943    alice
blue                f1c38dbaeeb473c36716f6494d803fbfbe9d8a76916f7c0093f227821e378197    alice, bob

List of signers and their keys for example/trust-demo:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	ecc457614c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

When `alice`, one of the signers, runs `docker trust revoke`:

```bash
$ docker trust revoke example/trust-demo
Please confirm you would like to delete all signature data for example/trust-demo? [y/N] y
Enter passphrase for delegation key with ID 27d42a8:
Successfully deleted signature for example/trust-demo
```

All tags that have `alice`'s signature on them are removed from the list of released tags:

```bash
$ docker trust view example/trust-demo

No signatures for example/trust-demo


List of signers and their keys for example/trust-demo:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	ecc457614c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

