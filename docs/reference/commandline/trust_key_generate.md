---
title: "key generate"
description: "The key generate command description and usage"
keywords: "key, notary, trust"
---

<!-- This file is maintained within the docker/cli Github
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# trust key generate

```markdown
Usage:  docker trust key generate NAME

Generate and load a signing key-pair

Options:
      --dir string   Directory to generate key in, defaults to current directory
      --help         Print usage
```

## Description

`docker trust key generate` generates a key-pair to be used with signing,
 and loads the private key into the local docker trust keystore.

## Examples

### Generate a key-pair

```bash
$ docker trust key generate alice

Generating key for alice...
Enter passphrase for new alice key with ID 17acf3c:
Repeat passphrase for new alice key with ID 17acf3c:
Successfully generated and loaded private key. Corresponding public key available: alice.pub
$ ls
alice.pub

```

The private signing key is encrypted by the passphrase and loaded into the docker trust keystore.
All passphrase requests to sign with the key will be referred to by the provided `NAME`.

The public key component `alice.pub` will be available in the current working directory, and can
be used directly by `docker trust signer add`.

Provide the `--dir` argument to specify a directory to generate the key in:

```bash
$ docker trust key generate alice --dir /foo

Generating key for alice...
Enter passphrase for new alice key with ID 17acf3c:
Repeat passphrase for new alice key with ID 17acf3c:
Successfully generated and loaded private key. Corresponding public key available: alice.pub
$ ls /foo
alice.pub

```
