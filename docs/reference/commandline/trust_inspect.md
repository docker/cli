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
Usage:  docker trust inspect IMAGE[:TAG]

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
```

The `SignedTags` key will list the `SignedTag` name, its `Digest`, and the `Signers` responsible for the signature.

`AdministrativeKeys` will list the `Repository` and `Root` keys.

This format mirrors the output of `docker trust view` 

If signers are set up for the repository via other `docker trust` commands, `docker trust inspect` includes a `Signers` key:

```bash

$ docker trust inspect my-image:purple | jq
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
```

If the image tag is unsigned or unavailable, `docker trust inspect` does not display any signed tags.

```bash
$ docker trust inspect unsigned-img
No signatures or cannot access unsigned-img
```

However, if other tags are signed in the same image repository, `docker trust inspect` reports relevant key information and omits the `SignedTags` key.

```bash
$ docker trust inspect alpine:unsigned | jq
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
```

### Get details about signatures for all image tags in a repository

```bash
$ docker trust inspect alpine | jq
{
  "SignedTags": [
    {
      "SignedTag": "2.6",
      "Digest": "9ace551613070689a12857d62c30ef0daa9a376107ec0fff0e34786cedb3399b",
      "Signers": [
        "Repo Admin"
      ]
    },
    {
      "SignedTag": "2.7",
      "Digest": "9f08005dff552038f0ad2f46b8e65ff3d25641747d3912e3ea8da6785046561a",
      "Signers": [
        "Repo Admin"
      ]
    },
    {
      "SignedTag": "3.1",
      "Digest": "2d74cbc2fbe3d261fdcca45d493ce1e3f3efd270114a62e383a8e45caeb48788",
      "Signers": [
        "Repo Admin"
      ]
    },
    {
      "SignedTag": "3.2",
      "Digest": "8565a58be8238ef688dbd90e43ec8e080114f1e1db846399116543eb8ef7d7b7",
      "Signers": [
        "Repo Admin"
      ]
    },
    {
      "SignedTag": "3.3",
      "Digest": "06fa785d55c35050241c60274e24ad57025683d5e939b3a31cc94193ca24740b",
      "Signers": [
        "Repo Admin"
      ]
    },
    {
      "SignedTag": "3.4",
      "Digest": "915b0ffca1d76ac57d83f28d568bcb516b6c274843ea8df7fac4b247440f796b",
      "Signers": [
        "Repo Admin"
      ]
    },
    {
      "SignedTag": "3.5",
      "Digest": "b007a354427e1880de9cdba533e8e57382b7f2853a68a478a17d447b302c219c",
      "Signers": [
        "Repo Admin"
      ]
    },
    {
      "SignedTag": "3.6",
      "Digest": "d6bfc3baf615dc9618209a8d607ba2a8103d9c8a405b3bd8741d88b4bef36478",
      "Signers": [
        "Repo Admin"
      ]
    },
    {
      "SignedTag": "edge",
      "Digest": "23e7d843e63a3eee29b6b8cfcd10e23dd1ef28f47251a985606a31040bf8e096",
      "Signers": [
        "Repo Admin"
      ]
    },
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
```
