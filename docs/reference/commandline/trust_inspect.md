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

`docker trust inspect` prints the trust information in a machine-readable format. Refer to
[`docker trust view`](trust_view.md) for a human-friendly output.

`docker trust inspect` is currently experimental.


## Examples

### Get low-level details about signatures for a single image tag

Use the `docker trust inspect` to get trust information about an image. The
following example prints trust information for the `alpine:latest` image:

```bash
$ docker trust inspect alpine:latest
[
  {
    "Name": "alpine:latest",
    "SignedTags": [
      {
        "SignedTag": "latest",
        "Digest": "d6bfc3baf615dc9618209a8d607ba2a8103d9c8a405b3bd8741d88b4bef36478",
        "Signers": [
          "Repo Admin"
        ]
      }
    ],
    "Signers": [],
    "AdminstrativeKeys": [
      {
        "Name": "Repository",
        "Keys": [
            {
                "ID": "5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd"
            }
        ]
      },
      {
        "Name": "Root",
        "Keys": [
            {
                "ID": "a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce"
            }
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
$ docker trust inspect my-image:purple
[
  {
    "Name": "my-image:purple",
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
            {
                "ID": "04dd031411ed671ae1e12f47ddc8646d98f135090b01e54c3561e843084484a3"
            },
            {
                "ID": "6a11e4898a4014d400332ab0e096308c844584ff70943cdd1d6628d577f45fd8"
            }
        ]
      },
      {
        "Name": "bob",
        "Keys": [
            {
                "ID": "433e245c656ae9733cdcc504bfa560f90950104442c4528c9616daa45824ccba"
            }
        ]
      },
      {
        "Name": "carol",
        "Keys": [
            {
                "ID": "d32fa8b5ca08273a2880f455fcb318da3dc80aeae1a30610815140deef8f30d9"
            },
            {
                "ID": "9a8bbec6ba2af88a5fad6047d428d17e6d05dbdd03d15b4fc8a9a0e8049cd606"
            }
        ]
      }
    ],
    "AdminstrativeKeys": [
      {
        "Name": "Repository",
        "Keys": [
            {
                "ID": "27df2c8187e7543345c2e0bf3a1262e0bc63a72754e9a7395eac3f747ec23a44"
            }
        ]
      },
      {
        "Name": "Root",
        "Keys": [
            {
                "ID": "40b66ccc8b176be8c7d365a17f3e046d1c3494e053dd57cfeacfe2e19c4f8e8f"
            }
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

However, if other tags are signed in the same image repository, `docker trust inspect` reports relevant key information:

```bash
$ docker trust inspect alpine:unsigned
[
  {
    "Name": "alpine:unsigned",
    "Signers": [],
    "AdminstrativeKeys": [
      {
        "Name": "Repository",
        "Keys": [
            {
                "ID": "5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd"
            }
        ]
      },
      {
        "Name": "Root",
        "Keys": [
            {
                "ID": "a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce"
            }
        ]
      }
    ]
  }
]
```

### Get details about signatures for all image tags in a repository

If no tag is specified, `docker trust inspect` will report details for all signed tags in the repository:

```bash
$ docker trust inspect alpine
[
    {
        "Name": "alpine",
        "SignedTags": [
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
        "Signers": [],
        "AdminstrativeKeys": [
            {
                "Name": "Repository",
                "Keys": [
                    {
                        "ID": "5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd"
                    }
                ]
            },
            {
                "Name": "Root",
                "Keys": [
                    {
                        "ID": "a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce"
                    }
                ]
            }
        ]
    }
]
```


### Get details about signatures for multiple images

`docker trust inspect` can take multiple repositories and images as arguments, and reports the results in an ordered list:

```bash
$ docker trust inspect alpine notary
[
    {
        "Name": "alpine",
        "SignedTags": [
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
                "SignedTag": "integ-test-base",
                "Digest": "3952dc48dcc4136ccdde37fbef7e250346538a55a0366e3fccc683336377e372",
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
        "Signers": [],
        "AdminstrativeKeys": [
            {
                "Name": "Repository",
                "Keys": [
                    {
                        "ID": "5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd"
                    }
                ]
            },
            {
                "Name": "Root",
                "Keys": [
                    {
                        "ID": "a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce"
                    }
                ]
            }
        ]
    },
    {
        "Name": "notary",
        "SignedTags": [
            {
                "SignedTag": "server",
                "Digest": "71f64ab718a3331dee103bc5afc6bc492914738ce37c2d2f127a8133714ecf5c",
                "Signers": [
                    "Repo Admin"
                ]
            },
            {
                "SignedTag": "signer",
                "Digest": "a6122d79b1e74f70b5dd933b18a6d1f99329a4728011079f06b245205f158fe8",
                "Signers": [
                    "Repo Admin"
                ]
            }
        ],
        "Signers": [],
        "AdminstrativeKeys": [
            {
                "Name": "Root",
                "Keys": [
                    {
                        "ID": "8cdcdef5bd039f4ab5a029126951b5985eebf57cabdcdc4d21f5b3be8bb4ce92"
                    }
                ]
            },
            {
                "Name": "Repository",
                "Keys": [
                    {
                        "ID": "85bfd031017722f950d480a721f845a2944db26a3dc084040a70f1b0d9bbb3df"
                    }
                ]
            }
        ]
    }
]
```
