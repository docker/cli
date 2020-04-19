---
title: "secret create"
description: "The secret create command description and usage"
keywords: ["secret, create"]
---

# secret create

```Markdown
Usage:	docker secret create [OPTIONS] SECRET [file|-]

Create a secret from a file or STDIN as content

Options:
  -l, --label list               Secret labels
      --template-driver string   Template driver
```

## Description

Creates a secret using standard input or from a file for the secret content.

For detailed information about using secrets, refer to [manage sensitive data with Docker secrets](https://docs.docker.com/engine/swarm/secrets/).

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### Create a secret

```bash
$ printf <secret> | docker secret create my_secret -

onakdyv307se2tl7nl20anokv

$ docker secret ls

ID                          NAME                CREATED             UPDATED
onakdyv307se2tl7nl20anokv   my_secret           6 seconds ago       6 seconds ago
```

### Create a secret with a file

```bash
$ docker secret create my_secret ./secret.json

dg426haahpi5ezmkkj5kyl3sn

$ docker secret ls

ID                          NAME                CREATED             UPDATED
dg426haahpi5ezmkkj5kyl3sn   my_secret           7 seconds ago       7 seconds ago
```

### Create a secret with labels

```bash
$ docker secret create --label env=dev \
                       --label rev=20170324 \
                       my_secret ./secret.json

eo7jnzguqgtpdah3cm5srfb97
```

```none
$ docker secret inspect my_secret

[
    {
        "ID": "eo7jnzguqgtpdah3cm5srfb97",
        "Version": {
            "Index": 17
        },
        "CreatedAt": "2017-03-24T08:15:09.735271783Z",
        "UpdatedAt": "2017-03-24T08:15:09.735271783Z",
        "Spec": {
            "Name": "my_secret",
            "Labels": {
                "env": "dev",
                "rev": "20170324"
            }
        }
    }
]
```


## Related commands

* [secret inspect](secret_inspect.md)
* [secret ls](secret_ls.md)
* [secret rm](secret_rm.md)
