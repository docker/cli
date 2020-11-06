---
title: "stack rm"
description: "The stack rm command description and usage"
keywords: "stack, rm, remove, down"
---

# stack rm

```markdown
Usage:  docker stack rm [OPTIONS] STACK [STACK...]

Remove one or more stacks

Aliases:
  rm, remove, down

Options:
      --help                  Print usage
      --kubeconfig string     Kubernetes config file
      --namespace string      Kubernetes namespace to use
      --orchestrator string   Orchestrator to use (swarm|kubernetes|all)
```

## Description

Remove the stack from the swarm.

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### Remove a stack

This will remove the stack with the name `myapp`. Services, networks, and secrets associated with the stack will be removed.

```bash
$ docker stack rm myapp

Removing service myapp_redis
Removing service myapp_web
Removing service myapp_lb
Removing network myapp_default
Removing network myapp_frontend
```

### Remove multiple stacks

This will remove all the specified stacks, `myapp` and `vossibility`. Services, networks, and secrets associated with all the specified stacks will be removed.

```bash
$ docker stack rm myapp vossibility

Removing service myapp_redis
Removing service myapp_web
Removing service myapp_lb
Removing network myapp_default
Removing network myapp_frontend
Removing service vossibility_nsqd
Removing service vossibility_logstash
Removing service vossibility_elasticsearch
Removing service vossibility_kibana
Removing service vossibility_ghollector
Removing service vossibility_lookupd
Removing network vossibility_default
Removing network vossibility_vossibility
```

## Related commands

* [stack deploy](stack_deploy.md)
* [stack ls](stack_ls.md)
* [stack ps](stack_ps.md)
* [stack services](stack_services.md)
