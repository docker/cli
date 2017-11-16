---
title: "trust inspect"
description: "The inspect command description and usage"
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

# trust inspect

```markdown
Usage:  docker trust inspect IMAGE[:TAG] [IMAGE[:TAG]...]

Return low-level information about keys and signatures

```

## Description

`docker trust inspect` provides low-level JSON information on signed repositories.
This includes all image tags that are signed, who signed them, and who can sign
new tags.

`docker trust inspect` is intended to be used for integrations into other systems, whereas `docker trust view` provides human-friendly output.

`docker trust inspect` is currently experimental.


## Examples

### Get low-level details about signatures for a single image tag


```bash
$ docker trust inspect alpine:latest | jq
[
  {
    "SignedTags": [
      {
        "SignedTag": "latest",
        "Digest": "d6bfc3baf615dc9618209a8d607ba2a8103d9c8a405b3bd8741d88b4bef36478",
        "Signers": [
          "Repo Admin"
        ]
      }
    ],
    "AdminstrativeKeys": [
      {
        "Name": "Repository",
        "Keys": [
          "5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd"
        ]
      },
      {
        "Name": "Root",
        "Keys": [
          "a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce"
        ]
      }
    ]
  }
]
```

The `SignedTags` key will list the `SignedTag` name, its `Digest`, and the `Signers` responsible for the signature.

`AdministrativeKeys` will list the `Repository` and `Root` keys.

This format mirrors the output of `docker trust view` 

If signers are set up for the repository via other `docker trust` commands, `docker trust inspect` includes a `Signers` key:

```bash

$ docker trust inspect my-image:purple | jq
[
  {
    "SignedTags": [
      {
        "SignedTag": "purple",
        "Digest": "941d3dba358621ce3c41ef67b47cf80f701ff80cdf46b5cc86587eaebfe45557",
        "Signers": [
          "alice",
          "bob",
          "carol"
        ]
      }
    ],
    "Signers": [
      {
        "Name": "alice",
        "Keys": [
          "04dd031411ed671ae1e12f47ddc8646d98f135090b01e54c3561e843084484a3",
          "6a11e4898a4014d400332ab0e096308c844584ff70943cdd1d6628d577f45fd8"
        ]
      },
      {
        "Name": "bob",
        "Keys": [
          "433e245c656ae9733cdcc504bfa560f90950104442c4528c9616daa45824ccba"
        ]
      },
      {
        "Name": "carol",
        "Keys": [
          "d32fa8b5ca08273a2880f455fcb318da3dc80aeae1a30610815140deef8f30d9",
          "9a8bbec6ba2af88a5fad6047d428d17e6d05dbdd03d15b4fc8a9a0e8049cd606"
        ]
      }
    ],
    "AdminstrativeKeys": [
      {
        "Name": "Repository",
        "Keys": [
          "27df2c8187e7543345c2e0bf3a1262e0bc63a72754e9a7395eac3f747ec23a44"
        ]
      },
      {
        "Name": "Root",
        "Keys": [
          "40b66ccc8b176be8c7d365a17f3e046d1c3494e053dd57cfeacfe2e19c4f8e8f"
        ]
      }
    ]
  }
]
```

If the image tag is unsigned or unavailable, `docker trust inspect` does not display any signed tags.

```bash
$ docker trust inspect unsigned-img
No signatures or cannot access unsigned-img
```

However, if other tags are signed in the same image repository, `docker trust inspect` reports relevant key information and omits the `SignedTags` key.

```bash
$ docker trust inspect alpine:unsigned | jq
[
  {
    "AdminstrativeKeys": [
      {
        "Name": "Repository",
        "Keys": [
          "5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd"
        ]
      },
      {
        "Name": "Root",
        "Keys": [
          "a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce"
        ]
      }
    ]
  }
]
```

### Get details about signatures for all image tags in a repository

```bash
$ docker trust inspect alpine | jq

```


### Get details about signatures for multiple images

```bash
$ docker trust inspect alpine ubuntu | jq
```
