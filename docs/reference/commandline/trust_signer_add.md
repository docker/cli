---
title: "signer add"
description: "The signer add command description and usage"
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

# trust signer add

```markdown
Usage:	docker trust signer add [OPTIONS] NAME REPOSITORY [REPOSITORY...]

Add a signer

Options:
      --help       Print usage
  -k, --key list   Path to the signer's public key file
```

## Description

`docker trust signer add` adds signers to signed repositories.

## Examples

### Add a signer to a repo

To add a new signer, `alice`, to this repository: 

```bash
$ docker trust view example/trust-demo

No signatures for example/trust-demo


List of signers and their keys:

SIGNER              KEYS
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	642692c14c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

Add `alice` with `docker trust signer add`:

```bash
$ docker trust signer add alice example/trust-demo --key alice.crt
  Adding signer "alice" to example/trust-demo...
  Enter passphrase for repository key with ID 642692c: 
  Successfully added signer: alice to example/trust-demo
```

`docker trust view` now lists `alice` as a valid signer:

```bash
$ docker trust view example/trust-demo

No signatures for example/trust-demo


List of signers and their keys:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	642692c14c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

## Initialize a new repo and add a signer

When adding a signer on a repo for the first time, `docker trust signer add` sets up a new repo if it doesn't exist.

```bash
$ docker trust view example/trust-demo
No signatures or cannot access example/trust-demo
```

```bash
$ docker trust signer add alice example/trust-demo --key alice.crt
 Initializing signed repository for example/trust-demo...
 Enter passphrase for root key with ID 748121c: 
 Enter passphrase for new repository key with ID 95b9e55: 
 Repeat passphrase for new repository key with ID 95b9e55: 
 Successfully initialized "example/trust-demo"
 
 Adding signer "alice" to example/trust-demo...
 Successfully added signer: alice to example/trust-demo
```

```bash
$ docker trust view example/trust-demo

No signatures for example/trust-demo


SIGNED TAG          DIGEST                                                             SIGNERS

List of signers and their keys:

SIGNER              KEYS
alice               6d52b29d940f

Administrative keys for example/trust-demo:
Repository Key:	95b9e5565eac3ef5ec01406801bdfb70feb40c17808d2222427c18046eb63beb
Root Key:	748121c14bd1461f6c58cb3ef39087c8fdc7633bb11a98af844fd9a04e208103
```

## Add a signer to multiple repos
To add a signer, `alice`, to multiple repositories: 

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
Add `alice` to both repositories with a single `docker trust signer add` command:

```bash
$ docker trust signer add alice example/trust-demo example/trust-demo2 --key alice.crt
Adding signer "alice" to example/trust-demo...
Enter passphrase for repository key with ID 95b9e55: 
Successfully added signer: alice to example/trust-demo

Adding signer "alice" to example/trust-demo2...
Enter passphrase for repository key with ID ece554f: 
Successfully added signer: alice to example/trust-demo2
```
`docker trust view` now lists `alice` as a valid signer of both `example/trust-demo` and `example/trust-demo2`:


```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                             SIGNERS
v1                  74d4bfa917d55d53c7df3d2ab20a8d926874d61c3da5ef6de15dd2654fc467c4   bob

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
v1                  74d4bfa917d55d53c7df3d2ab20a8d926874d61c3da5ef6de15dd2654fc467c4   bob

List of signers and their keys:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo2:
Repository Key:	ece554f14c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4553d2ab20a8d9268
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```


`docker trust signer add` adds signers to repositories on a best effort basis, so it will continue to add the signer to subsequent repositories if one attempt fails:

```bash
$ docker trust signer add alice example/unauthorized example/authorized --key alice.crt
Adding signer "alice" to example/unauthorized...
you are not authorized to perform this operation: server returned 401.

Adding signer "alice" to example/authorized...
Enter passphrase for repository key with ID c6772a0: 
Successfully added signer: alice to example/authorized

Failed to add signer to: example/unauthorized
```
