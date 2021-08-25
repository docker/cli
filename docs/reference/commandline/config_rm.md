---
title: "config rm"
description: "The config rm command description and usage"
keywords: ["config, rm"]
---

# config rm

```Markdown
Usage:  docker config rm CONFIG [CONFIG...]

Remove one or more configs

Aliases:
  rm, remove

Options:
      --help   Print usage
```

## Description

Removes the specified configs from the swarm.

For detailed information about using configs, refer to [store configuration data using Docker Configs](https://docs.docker.com/engine/swarm/configs/).

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

This example removes a config:

```console
$ docker config rm my_config
sapth4csdo5b6wz2p5uimh5xg
```

> **Warning**
>
> Unlike `docker rm`, this command does not ask for confirmation before removing
> a config.


## Related commands

* [config create](config_create.md)
* [config inspect](config_inspect.md)
* [config ls](config_ls.md)
