---
title: "swarm update"
description: "The swarm update command description and usage"
keywords: "swarm, update"
---

# swarm update

```markdown
Usage:  docker swarm update [OPTIONS]

Update the swarm

Options:
      --autolock                        Change manager autolocking setting (true|false)
      --cert-expiry duration            Validity period for node certificates (ns|us|ms|s|m|h) (default 2160h0m0s)
      --dispatcher-heartbeat duration   Dispatcher heartbeat period (ns|us|ms|s|m|h) (default 5s)
      --external-ca external-ca         Specifications of one or more certificate signing endpoints
      --help                            Print usage
      --max-snapshots uint              Number of additional Raft snapshots to retain
      --snapshot-interval uint          Number of log entries between Raft snapshots (default 10000)
      --task-history-limit int          Task history retention limit (default 5)
```

## Description

Updates a swarm with new parameter values.

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

```bash
$ docker swarm update --cert-expiry 720h
```

## Related commands

* [swarm ca](swarm_ca.md)
* [swarm init](swarm_init.md)
* [swarm join](swarm_join.md)
* [swarm join-token](swarm_join-token.md)
* [swarm leave](swarm_leave.md)
* [swarm unlock](swarm_unlock.md)
* [swarm unlock-key](swarm_unlock-key.md)
