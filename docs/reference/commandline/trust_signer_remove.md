---
title: "signer remove"
description: "The signer remove command description and usage"
keywords: "signer, notary, trust"
---

<!-- This file is maintained within the docker/cli Github
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# trust signer remove

```markdown
Usage:	docker trust signer remove [OPTIONS] NAME REPOSITORY [REPOSITORY...]

Remove a signer

Options:
  -f, --force   Do not prompt for confirmation before removing the most recent signer
  --help        Print usage
```

## Description

`docker trust signer remove` removes signers from signed repositories.

## Examples

### Remove a signer from a repo

To remove an existing signer, `alice`, from this repository: 

```bash
$ docker trust view example/trust-demo

No signatures for example/trust-demo


List of signers and their keys:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	ecc457614c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

Remove `alice` with `docker trust signer remove`:

```bash
$ docker trust signer remove alice example/trust-demo
  Removing signer "alice" from image example/trust-demo...
  Enter passphrase for repository key with ID 642692c: 
  Successfully removed alice from example/trust-demo

```

`docker trust view` now does not list `alice` as a valid signer:

```bash
$ docker trust view example/trust-demo

No signatures for example/trust-demo


List of signers and their keys:

SIGNER              KEYS
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	ecc457614c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

### Remove a signer from multiple repos

To remove an existing signer, `alice`, from multiple repositories: 

```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                             SIGNERS
v1                  74d4bfa917d55d53c7df3d2ab20a8d926874d61c3da5ef6de15dd2654fc467c4   alice, bob

List of signers and their keys:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	95b9e5514c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```
```bash
$ docker trust view example/trust-demo2
SIGNED TAG          DIGEST                                                             SIGNERS
v1                  74d4bfa917d55d53c7df3d2ab20a8d926874d61c3da5ef6de15dd2654fc467c4   alice, bob

List of signers and their keys:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo2:
Repository Key:	ece554f14c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4553d2ab20a8d9268
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```
Remove `alice` from both images with a single `docker trust signer remove` command:

```bash
$ docker trust signer remove alice example/trust-demo example/trust-demo2
Removing signer "alice" from image example/trust-demo...
Enter passphrase for repository key with ID 95b9e55: 
Successfully removed alice from example/trust-demo

Removing signer "alice" from image example/trust-demo2...
Enter passphrase for repository key with ID ece554f: 
Successfully removed alice from example/trust-demo2
```
`docker trust view` no longer lists `alice` as a valid signer of either `example/trust-demo` or `example/trust-demo2`:
```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                             SIGNERS
v1                  74d4bfa917d55d53c7df3d2ab20a8d926874d61c3da5ef6de15dd2654fc467c4   bob

List of signers and their keys:

SIGNER              KEYS
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	ecc457614c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```
```bash
$ docker trust view example/trust-demo2
SIGNED TAG          DIGEST                                                             SIGNERS
v1                  74d4bfa917d55d53c7df3d2ab20a8d926874d61c3da5ef6de15dd2654fc467c4   bob

List of signers and their keys:

SIGNER              KEYS
bob                 5600f5ab76a2

Administrative keys for example/trust-demo2:
Repository Key:	ece554f14c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4553d2ab20a8d9268
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

`docker trust signer remove` removes signers to repositories on a best effort basis, so it will continue to remove the signer from subsequent repositories if one attempt fails:

```bash
$ docker trust signer remove alice example/unauthorized example/authorized
Removing signer "alice" from image example/unauthorized...
No signer alice for image example/unauthorized

Removing signer "alice" from image example/authorized...
Enter passphrase for repository key with ID c6772a0: 
Successfully removed alice from example/authorized

Error removing signer from: example/unauthorized
```

