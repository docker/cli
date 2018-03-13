---
title: "trust inspect"
description: "The inspect command description and usage"
keywords: "inspect, notary, trust"
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

Options:
      --help            Print usage
      --pretty          Print the information in a human friendly format
```

## Description

`docker trust inspect` provides low-level JSON information on signed repositories.
This includes all image tags that are signed, who signed them, and who can sign
new tags.

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

The `SignedTags` key will list the `SignedTag` name, its `Digest`,
and the `Signers` responsible for the signature.

`AdministrativeKeys` will list the `Repository` and `Root` keys.

If signers are set up for the repository via other `docker trust`
commands, `docker trust inspect` includes a `Signers` key:

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

If the image tag is unsigned or unavailable, `docker trust inspect` does not
display any signed tags.

```bash
$ docker trust inspect unsigned-img
No signatures or cannot access unsigned-img
```

However, if other tags are signed in the same image repository,
`docker trust inspect` reports relevant key information:

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

If no tag is specified, `docker trust inspect` will report details for all
signed tags in the repository:

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

`docker trust inspect` can take multiple repositories and images as arguments,
and reports the results in an ordered list:

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

### Formatting

You can print the inspect output in a human-readable format instead of the default
JSON output, by using the `--pretty` option:

### Get details about signatures for a single image tag

```bash
$ docker trust inspect --pretty alpine:latest

SIGNED TAG          DIGEST                                                             SIGNERS
latest              1072e499f3f655a032e88542330cf75b02e7bdf673278f701d7ba61629ee3ebe   (Repo Admin)

Administrative keys for alpine:latest:
Repository Key:	5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd
Root Key:	a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce
```

The `SIGNED TAG` is the signed image tag with a unique content-addressable
`DIGEST`. `SIGNERS` lists all entities who have signed.

The administrative keys listed specify the root key of trust, as well as
the administrative repository key. These keys are responsible for modifying
signers, and rotating keys for the signed repository.

If signers are set up for the repository via other `docker trust` commands,
`docker trust inspect --pretty` displays them appropriately as a `SIGNER`
and specify their `KEYS`:

```bash
$ docker trust inspect --pretty my-image:purple
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

However, if other tags are signed in the same image repository,
`docker trust inspect` reports relevant key information.

```bash
$ docker trust inspect --pretty alpine:unsigned

No signatures for alpine:unsigned


Administrative keys for alpine:unsigned:
Repository Key:	5a46c9aaa82ff150bb7305a2d17d0c521c2d784246807b2dc611f436a69041fd
Root Key:	a2489bcac7a79aa67b19b96c4a3bf0c675ffdf00c6d2fabe1a5df1115e80adce
```

### Get details about signatures for all image tags in a repository

```bash
$ docker trust inspect --pretty alpine
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
$ docker trust inspect --pretty my-image
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
