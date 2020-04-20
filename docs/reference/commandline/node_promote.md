---
title: "node promote"
description: "The node promote command description and usage"
keywords: "node, promote"
---

# node promote

```markdown
Usage:  docker node promote NODE [NODE...]

Promote one or more nodes to manager in the swarm

Options:
      --help   Print usage
```

## Description

Promotes a node to manager. This command can only be executed on a manager node.

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

```bash
$ docker node promote <node name>
```

## Related commands

* [node demote](node_demote.md)
* [node inspect](node_inspect.md)
* [node ls](node_ls.md)
* [node ps](node_ps.md)
* [node rm](node_rm.md)
* [node update](node_update.md)
