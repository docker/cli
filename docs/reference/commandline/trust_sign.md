---
title: "trust sign"
description: "The sign command description and usage"
keywords: "sign, notary, trust"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# trust sign

```markdown
Usage:  docker trust sign [OPTIONS] IMAGE:TAG

Sign an image

Options:
      --help    print usage
      --local   force the signing of a local image

```

## Description

`docker trust sign` adds signatures to tags to create signed repositories.

## Examples

### Sign a tag as a repo admin

Given an image:

```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                             SIGNERS
v1                  c24134c079c35e698060beabe110bb83ab285d0d978de7d92fed2c8c83570a41   (Repo Admin)

Administrative keys for example/trust-demo:
Repository Key:	36d4c3601102fa7c5712a343c03b94469e5835fb27c191b529c06fd19c14a942
Root Key:	246d360f7c53a9021ee7d4259e3c5692f3f1f7ad4737b1ea8c7b8da741ad980b
```

Sign a new tag with `docker trust sign`:

```bash
$ docker trust sign example/trust-demo:v2
Signing and pushing trust metadata for example/trust-demo:v2
The push refers to a repository [docker.io/example/trust-demo]
eed4e566104a: Layer already exists
77edfb6d1e3c: Layer already exists
c69f806905c2: Layer already exists
582f327616f1: Layer already exists
a3fbb648f0bd: Layer already exists
5eac2de68a97: Layer already exists
8d4d1ab5ff74: Layer already exists
v2: digest: sha256:8f6f460abf0436922df7eb06d28b3cdf733d2cac1a185456c26debbff0839c56 size: 1787
Signing and pushing trust metadata
Enter passphrase for repository key with ID 36d4c36:
Successfully signed docker.io/example/trust-demo:v2
```

`docker trust view` lists the new signature:

```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                             SIGNERS
v1                  c24134c079c35e698060beabe110bb83ab285d0d978de7d92fed2c8c83570a41   (Repo Admin)
v2                  8f6f460abf0436922df7eb06d28b3cdf733d2cac1a185456c26debbff0839c56   (Repo Admin)

Administrative keys for example/trust-demo:
Repository Key:	36d4c3601102fa7c5712a343c03b94469e5835fb27c191b529c06fd19c14a942
Root Key:	246d360f7c53a9021ee7d4259e3c5692f3f1f7ad4737b1ea8c7b8da741ad980b
```

### Sign a tag as a signer

Given an image:

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

Sign a new tag with `docker trust sign`:

```bash
$ docker trust sign example/trust-demo:v1
Signing and pushing trust metadata for example/trust-demo:v1
The push refers to a repository [docker.io/example/trust-demo]
26b126eb8632: Layer already exists
220d34b5f6c9: Layer already exists
8a5132998025: Layer already exists
aca233ed29c3: Layer already exists
e5d2f035d7a4: Layer already exists
v1: digest: sha256:74d4bfa917d55d53c7df3d2ab20a8d926874d61c3da5ef6de15dd2654fc467c4 size: 1357
Signing and pushing trust metadata
Enter passphrase for delegation key with ID 27d42a8:
Successfully signed docker.io/example/trust-demo:v1
```

`docker trust view` lists the new signature:

```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                             SIGNERS
v1                  74d4bfa917d55d53c7df3d2ab20a8d926874d61c3da5ef6de15dd2654fc467c4   alice

List of signers and their keys for example/trust-demo:

SIGNER              KEYS
alice               05e87edcaecb
bob                 5600f5ab76a2

Administrative keys for example/trust-demo:
Repository Key:	ecc457614c9fc399da523a5f4e24fe306a0a6ee1cc79a10e4555b3c6ab02f71e
Root Key:	3cb2228f6561e58f46dbc4cda4fcaff9d5ef22e865a94636f82450d1d2234949
```

## Initialize a new repo and sign a tag

When signing an image on a repo for the first time, `docker trust sign` sets up new keys before signing the image.

```bash
$ docker trust view example/trust-demo
No signatures or cannot access example/trust-demo
```

```bash
$ docker trust sign example/trust-demo:v1
Signing and pushing trust metadata for example/trust-demo:v1
Enter passphrase for root key with ID 36cac18:
Enter passphrase for new repository key with ID 731396b:
Repeat passphrase for new repository key with ID 731396b:
Enter passphrase for new alice key with ID 6d52b29:
Repeat passphrase for new alice key with ID 6d52b29:
Created signer: alice
Finished initializing "docker.io/example/trust-demo"
The push refers to a repository [docker.io/example/trust-demo]
eed4e566104a: Layer already exists
77edfb6d1e3c: Layer already exists
c69f806905c2: Layer already exists
582f327616f1: Layer already exists
a3fbb648f0bd: Layer already exists
5eac2de68a97: Layer already exists
8d4d1ab5ff74: Layer already exists
v1: digest: sha256:8f6f460abf0436922df7eb06d28b3cdf733d2cac1a185456c26debbff0839c56 size: 1787
Signing and pushing trust metadata
Enter passphrase for alice key with ID 6d52b29:
Successfully signed docker.io/example/trust-demo:v1
```

```bash
$ docker trust view example/trust-demo
SIGNED TAG          DIGEST                                                             SIGNERS
v1                  8f6f460abf0436922df7eb06d28b3cdf733d2cac1a185456c26debbff0839c56   alice

List of signers and their keys for example/trust-demo:

SIGNER              KEYS
alice               6d52b29d940f

Administrative keys for example/trust-demo:
Repository Key:	731396b65eac3ef5ec01406801bdfb70feb40c17808d2222427c18046eb63beb
Root Key:	70d174714bd1461f6c58cb3ef39087c8fdc7633bb11a98af844fd9a04e208103
```

