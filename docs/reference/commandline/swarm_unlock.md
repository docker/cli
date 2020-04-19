---
title: "swarm unlock"
description: "The swarm unlock command description and usage"
keywords: "swarm, unlock"
---

# swarm unlock

```markdown
Usage:	docker swarm unlock

Unlock swarm

Options:
      --help   Print usage
```

## Description

Unlocks a locked manager using a user-supplied unlock key. This command must be
used to reactivate a manager after its Docker daemon restarts if the autolock
setting is turned on. The unlock key is printed at the time when autolock is
enabled, and is also available from the `docker swarm unlock-key` command.

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

```bash
$ docker swarm unlock
Please enter unlock key:
```

## Related commands

* [swarm ca](swarm_ca.md)
* [swarm init](swarm_init.md)
* [swarm join](swarm_join.md)
* [swarm join-token](swarm_join-token.md)
* [swarm leave](swarm_leave.md)
* [swarm unlock-key](swarm_unlock-key.md)
* [swarm update](swarm_update.md)
