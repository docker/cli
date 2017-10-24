---
title: "config create"
description: "The config create command description and usage"
keywords: ["config, create"]
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# config create

```Markdown
Usage:	docker config create [OPTIONS] config file|-

Create a config from a file, directory or STDIN

Options:
      --help          Print usage
  -l, --label list    config labels (default [])
```

## Description

Creates a configuration using standard input, from a file for the config content or from a directory. You must run this command on a manager node. 

Specifying a directory will create a configuration file for every regular file found on the directory, ignoring any other entry types (e.g. subdirectories, symlinks, etc). Files names will be used as configuration names.

For detailed information about using configs, refer to [Swarm configuration files](https://docs.docker.com/engine/swarm/configs/).

## Examples

### Create a configuration

```bash
$ echo <config> | docker config create my_config -

onakdyv307se2tl7nl20anokv

$ docker config ls

ID                          NAME                CREATED             UPDATED
onakdyv307se2tl7nl20anokv   my_config           6 seconds ago       6 seconds ago
```

### Create a configuration with a file

```bash
$ docker config create my_config ./config.json

dg426haahpi5ezmkkj5kyl3sn

$ docker config ls

ID                          NAME                CREATED             UPDATED
dg426haahpi5ezmkkj5kyl3sn   my_config           7 seconds ago       7 seconds ago
```

### Create configurations from a directory

```bash
$ docker config create  ./my-conf-dirs

dg426haahpi5ezmkkj5kyl3sn
5kyl3sndg426haahpi5ezmkkj

$ docker config ls

ID                          NAME                CREATED             UPDATED
dg426haahpi5ezmkkj5kyl3sn   config.yaml         5 seconds ago       5 seconds ago
5kyl3sndg426haahpi5ezmkkj   another_config.txt  7 seconds ago       7 seconds ago
```


### Create a configuration with labels

```bash
$ docker config create --label env=dev \
                       --label rev=20170324 \
                       my_config ./config.json

eo7jnzguqgtpdah3cm5srfb97
```

```none
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
            }
        }
    }
]
```


## Related commands

* [config inspect]
* [config ls]
* [config rm]
