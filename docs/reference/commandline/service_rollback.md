---
title: "service rollback"
description: "The service rollback command description and usage"
keywords: "service, rollback"
---

# service rollback

```markdown
Usage:	docker service rollback SERVICE

Revert changes to a service's configuration

Options:
  -d, --detach       Exit immediately instead of waiting for the service to converge (default true)
      --help         Print usage
  -q, --quiet        Suppress progress output
```

## Description

Roll back a specified service to its previous version from the swarm.

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### Roll back to the previous version of a service

Use the `docker service rollback` command to roll back to the previous version
of a service. After executing this command, the service is reverted to the
configuration that was in place before the most recent `docker service update`
command.

The following example creates a service with a single replica, updates the
service to use three replicas, and then rolls back the service to the
previous version, having one replica.

Create a service with a single replica:

```bash
$ docker service create --name my-service -p 8080:80 nginx:alpine
```

Confirm that the service is running with a single replica:

```bash
$ docker service ls

ID                  NAME                MODE                REPLICAS            IMAGE               PORTS
xbw728mf6q0d        my-service          replicated          1/1                 nginx:alpine        *:8080->80/tcp
```

Update the service to use three replicas:

```bash
$ docker service update --replicas=3 my-service

$ docker service ls

ID                  NAME                MODE                REPLICAS            IMAGE               PORTS
xbw728mf6q0d        my-service          replicated          3/3                 nginx:alpine        *:8080->80/tcp
```

Now roll back the service to its previous version, and confirm it is
running a single replica again:

```bash
$ docker service rollback my-service

$ docker service ls

ID                  NAME                MODE                REPLICAS            IMAGE               PORTS
xbw728mf6q0d        my-service          replicated          1/1                 nginx:alpine        *:8080->80/tcp
```

## Related commands

* [service create](service_create.md)
* [service inspect](service_inspect.md)
* [service logs](service_logs.md)
* [service ls](service_ls.md)
* [service ps](service_ps.md)
* [service rm](service_rm.md)
* [service scale](service_scale.md)
* [service update](service_update.md)
