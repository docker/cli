---
title: "secret rm"
description: "The secret rm command description and usage"
keywords: ["secret, rm"]
---

# secret rm

```Markdown
Usage:	docker secret rm SECRET [SECRET...]

Remove one or more secrets

Aliases:
  rm, remove

Options:
      --help   Print usage
```

## Description

Removes the specified secrets from the swarm.

For detailed information about using secrets, refer to [manage sensitive data with Docker secrets](https://docs.docker.com/engine/swarm/secrets/).

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

This example removes a secret:

```bash
$ docker secret rm secret.json
sapth4csdo5b6wz2p5uimh5xg
```

> **Warning**
>
> Unlike `docker rm`, this command does not ask for confirmation before removing
> a secret.


## Related commands

* [secret create](secret_create.md)
* [secret inspect](secret_inspect.md)
* [secret ls](secret_ls.md)
