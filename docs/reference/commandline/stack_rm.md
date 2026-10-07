# stack rm

<!---MARKER_GEN_START-->
Remove one or more stacks

### Aliases

`docker stack rm`, `docker stack remove`, `docker stack down`

### Options

| Name             | Type   | Default | Description                   |
|:-----------------|:-------|:--------|:------------------------------|
| `-d`, `--detach` | `bool` | `true`  | Do not wait for stack removal |


<!---MARKER_GEN_END-->

## Description

Remove the stack from the swarm.

By default, the command returns as soon as the removal of the stack's services,
networks, secrets, and configs has been requested; the tasks of the services
are stopped in the background. With `--detach=false`, the command waits for the
tasks to stop before it removes the networks of the stack, and returns once the
stack is gone. Removing the networks after the tasks have stopped lets the
swarm manager release every address the tasks held, including their addresses
on the `ingress` network.

> [!NOTE]
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### Remove a stack

This will remove the stack with the name `myapp`. Services, networks, and secrets
associated with the stack will be removed.

```console
$ docker stack rm myapp

Removing service myapp_redis
Removing service myapp_web
Removing service myapp_lb
Removing network myapp_default
Removing network myapp_frontend
```

### Remove multiple stacks

This will remove all the specified stacks, `myapp` and `vossibility`. Services,
networks, and secrets associated with all the specified stacks will be removed.

```console
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
