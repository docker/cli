---
title: "node demote"
description: "The node demote command description and usage"
keywords: "node, demote"
---

# node demote

```markdown
Usage:  docker node demote NODE [NODE...]

Demote one or more nodes from manager in the swarm

Options:
      --help   Print usage

```

## Description

Demotes an existing manager so that it is no longer a manager.

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the [Swarm mode
> section](https://docs.docker.com/engine/swarm/) in the documentation.

## Examples

```bash
$ docker node demote <node name>
```

## Related commands

* [node inspect](node_inspect.md)
* [node ls](node_ls.md)
* [node promote](node_promote.md)
* [node ps](node_ps.md)
* [node rm](node_rm.md)
* [node update](node_update.md)
