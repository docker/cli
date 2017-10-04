---
title: "trust view"
description: "The view command description and usage"
keywords: "view, notary, trust"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# trust view

```markdown
Usage:  docker trust view IMAGE[:TAG]

Display detailed information about keys and signatures

```

## Description

`docker trust view` provides detailed information on signed repositories.
This includes all image tags that are signed, who signed them, and who can sign
new tags.

By default, `docker trust view` renders results in a table.

`docker trust view` is currently experimental.


## Examples

### Get details about signatures for a single image tag


```bash
$ docker trust view alpine:latest

SIGNED TAG          DIGEST                                                             SIGNERS
latest              1072e499f3f655a032e88542330cf75b02e7bdf673278f701d7ba61629ee3ebe   (Repo Admin)

Administrative keys for alpine:latest:
Repository Key:	5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd
Root Key:	a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce
```

The `SIGNED TAG` is the signed image tag with a unique content-addressable `DIGEST`. `SIGNERS` lists all entities who have signed.

The administrative keys listed specify the root key of trust, as well as the administrative repository key. These keys are responsible for modifying signers, and rotating keys for the signed repository.

If signers are set up for the repository via other `docker trust` commands, `docker trust view` displays them appropriately as a `SIGNER` and specify their `KEYS`:

```bash
$ docker trust view my-image:purple
SIGNED TAG          DIGEST                                                              SIGNERS
purple              941d3dba358621ce3c41ef67b47cf80f701ff80cdf46b5cc86587eaebfe45557    alice, bob, carol

List of signers and their keys:

SIGNER              KEYS
alice               47caae5b3e61, a85aab9d20a4
bob                 034370bcbd77, 82a66673242c
carol               b6f9f8e1aab0

Administrative keys for my-image:
Repository Key:	27df2c8187e7543345c2e0bf3a1262e0bc63a72754e9a7395eac3f747ec23a44
Root Key:	40b66ccc8b176be8c7d365a17f3e046d1c3494e053dd57cfeacfe2e19c4f8e8f
```

If the image tag is unsigned or unavailable, `docker trust view` does not display any signed tags.

```bash
$ docker trust view unsigned-img
No signatures or cannot access unsigned-img
```

However, if other tags are signed in the same image repository, `docker trust view` reports relevant key information.

```bash
$ docker trust view alpine:unsigned

No signatures for alpine:unsigned


Administrative keys for alpine:unsigned:
Repository Key:	5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd
Root Key:	a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce
```

### Get details about signatures for all image tags in a repository

```bash
$ docker trust view alpine
SIGNED TAG          DIGEST                                                             SIGNERS
2.6                 9ace551613070689a12857d62c30ef0daa9a376107ec0fff0e34786cedb3399b   (Repo Admin)
2.7                 9f08005dff552038f0ad2f46b8e65ff3d25641747d3912e3ea8da6785046561a   (Repo Admin)
3.1                 d9477888b78e8c6392e0be8b2e73f8c67e2894ff9d4b8e467d1488fcceec21c8   (Repo Admin)
3.2                 19826d59171c2eb7e90ce52bfd822993bef6a6fe3ae6bb4a49f8c1d0a01e99c7   (Repo Admin)
3.3                 8fd4b76819e1e5baac82bd0a3d03abfe3906e034cc5ee32100d12aaaf3956dc7   (Repo Admin)
3.4                 833ad81ace8277324f3ca8c91c02bdcf1d13988d8ecf8a3f97ecdd69d0390ce9   (Repo Admin)
3.5                 af2a5bd2f8de8fc1ecabf1c76611cdc6a5f1ada1a2bdd7d3816e121b70300308   (Repo Admin)
3.6                 1072e499f3f655a032e88542330cf75b02e7bdf673278f701d7ba61629ee3ebe   (Repo Admin)
edge                79d50d15bd7ea48ea00cf3dd343b0e740c1afaa8e899bee475236ef338e1b53b   (Repo Admin)
latest              1072e499f3f655a032e88542330cf75b02e7bdf673278f701d7ba61629ee3ebe   (Repo Admin)

Administrative keys for alpine:
Repository Key:	5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd
Root Key:	a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce
```

Here's an example with signers that are set up by `docker trust` commands:

```bash
$ docker trust view my-image
SIGNED TAG          DIGEST                                                              SIGNERS
red                 852cc04935f930a857b630edc4ed6131e91b22073bcc216698842e44f64d2943    alice
blue                f1c38dbaeeb473c36716f6494d803fbfbe9d8a76916f7c0093f227821e378197    alice, bob
green               cae8fedc840f90c8057e1c24637d11865743ab1e61a972c1c9da06ec2de9a139    alice, bob
yellow              9cc65fc3126790e683d1b92f307a71f48f75fa7dd47a7b03145a123eaf0b45ba    carol
purple              941d3dba358621ce3c41ef67b47cf80f701ff80cdf46b5cc86587eaebfe45557    alice, bob, carol
orange              d6c271baa6d271bcc24ef1cbd65abf39123c17d2e83455bdab545a1a9093fc1c    alice

List of signers and their keys for my-image:

SIGNER              KEYS
alice               47caae5b3e61, a85aab9d20a4
bob                 034370bcbd77, 82a66673242c
carol               b6f9f8e1aab0

Administrative keys for my-image:
Repository Key:	27df2c8187e7543345c2e0bf3a1262e0bc63a72754e9a7395eac3f747ec23a44
Root Key:	40b66ccc8b176be8c7d365a17f3e046d1c3494e053dd57cfeacfe2e19c4f8e8f
```
