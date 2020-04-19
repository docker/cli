---
title: "secret inspect"
description: "The secret inspect command description and usage"
keywords: ["secret, inspect"]
---

# secret inspect

```Markdown
Usage:  docker secret inspect [OPTIONS] SECRET [SECRET...]

Display detailed information on one or more secrets

Options:
  -f, --format string   Format the output using the given Go template
      --help            Print usage
```

## Description

Inspects the specified secret.

By default, this renders all results in a JSON array. If a format is specified,
the given template will be executed for each result.

Go's [text/template](http://golang.org/pkg/text/template/) package
describes all the details of the format.

For detailed information about using secrets, refer to [manage sensitive data with Docker secrets](https://docs.docker.com/engine/swarm/secrets/).

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### Inspect a secret by name or ID

You can inspect a secret, either by its *name*, or *ID*

For example, given the following secret:

```bash
$ docker secret ls

ID                          NAME                CREATED             UPDATED
eo7jnzguqgtpdah3cm5srfb97   my_secret           3 minutes ago       3 minutes ago
```

```none
$ docker secret inspect secret.json

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

### Formatting

You can use the --format option to obtain specific information about a
secret. The following example command outputs the creation time of the
secret.

```bash
$ docker secret inspect --format='{{.CreatedAt}}' eo7jnzguqgtpdah3cm5srfb97

2017-03-24 08:15:09.735271783 +0000 UTC
```


## Related commands

* [secret create](secret_create.md)
* [secret ls](secret_ls.md)
* [secret rm](secret_rm.md)
