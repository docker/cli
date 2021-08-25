---
title: "config create"
description: "The config create command description and usage"
keywords: ["config, create"]
---

# config create

```Markdown
Usage:  docker config create [OPTIONS] CONFIG [file|-]

Create a config from a file or STDIN as content

Options:
  -l, --label list               Config labels
      --template-driver string   Template driver
```

## Description

Creates a config using standard input or from a file for the config content.

For detailed information about using configs, refer to [store configuration data using Docker Configs](https://docs.docker.com/engine/swarm/configs/).

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### Create a config

```console
$ printf <config> | docker config create my_config -

onakdyv307se2tl7nl20anokv

$ docker config ls

ID                          NAME                CREATED             UPDATED
onakdyv307se2tl7nl20anokv   my_config           6 seconds ago       6 seconds ago
```

### Create a config with a file

```console
$ docker config create my_config ./config.json

dg426haahpi5ezmkkj5kyl3sn

$ docker config ls

ID                          NAME                CREATED             UPDATED
dg426haahpi5ezmkkj5kyl3sn   my_config           7 seconds ago       7 seconds ago
```

### Create a config with labels

```console
$ docker config create \
    --label env=dev \
    --label rev=20170324 \
    my_config ./config.json

eo7jnzguqgtpdah3cm5srfb97
```

```console
$ docker config inspect my_config

[
    {
        "ID": "eo7jnzguqgtpdah3cm5srfb97",
        "Version": {
            "Index": 17
        },
        "CreatedAt": "2017-03-24T08:15:09.735271783Z",
        "UpdatedAt": "2017-03-24T08:15:09.735271783Z",
        "Spec": {
            "Name": "my_config",
            "Labels": {
                "env": "dev",
                "rev": "20170324"
            },
            "Data": "aGVsbG8K"
        }
    }
]
```


## Related commands

* [config inspect](config_inspect.md)
* [config ls](config_ls.md)
* [config rm](config_rm.md)
