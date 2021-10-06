---
title: "key load"
description: "The key load command description and usage"
keywords: "key, notary, trust"
---

# trust key load

```markdown
Usage:  docker trust key load [OPTIONS] KEYFILE

Load a private key file for signing

Options:
      --help          Print usage
      --name string   Name for the loaded key (default "signer")
```

## Description

`docker trust key load` adds private keys to the local docker trust keystore.

To add a signer to a repository use `docker trust signer add`.

## Examples

### Load a single private key

For a private key `alice.pem` with permissions `-rw-------`

```console
$ docker trust key load alice.pem

Loading key from "alice.pem"...
Enter passphrase for new signer key with ID f8097df:
Repeat passphrase for new signer key with ID f8097df:
Successfully imported key from alice.pem
```

To specify a name use the `--name` flag:

```console
$ docker trust key load --name alice-key alice.pem

Loading key from "alice.pem"...
Enter passphrase for new alice-key key with ID f8097df:
Repeat passphrase for new alice-key key with ID f8097df:
Successfully imported key from alice.pem
```
