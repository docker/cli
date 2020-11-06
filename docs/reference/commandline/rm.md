---
title: "rm"
description: "The rm command description and usage"
keywords: "remove, Docker, container"
---

# rm

```markdown
Usage:  docker rm [OPTIONS] CONTAINER [CONTAINER...]

Remove one or more containers

Options:
  -f, --force     Force the removal of a running container (uses SIGKILL)
      --help      Print usage
  -l, --link      Remove the specified link
  -v, --volumes   Remove anonymous volumes associated with the container
```

## Examples

### Remove a container

This removes the container referenced under the link `/redis`.

```bash
$ docker rm /redis

/redis
```

### Remove a link specified with `--link` on the default bridge network

This removes the underlying link between `/webapp` and the `/redis`
containers on the default bridge network, removing all network communication
between the two containers. This does not apply when `--link` is used with
user-specified networks.

```bash
$ docker rm --link /webapp/redis

/webapp/redis
```

### Force-remove a running container

This command force-removes a running container.

```bash
$ docker rm --force redis

redis
```

The main process inside the container referenced under the link `redis` will receive
`SIGKILL`, then the container will be removed.

### Remove all stopped containers

```bash
$ docker rm $(docker ps -a -q)
```

This command deletes all stopped containers. The command
`docker ps -a -q` above returns all existing container IDs and passes them to
the `rm` command which deletes them. Running containers are not deleted.

### Remove a container and its volumes

```bash
$ docker rm -v redis
redis
```

This command removes the container and any volumes associated with it.
Note that if a volume was specified with a name, it will not be removed.

### Remove a container and selectively remove volumes

```bash
$ docker create -v awesome:/foo -v /bar --name hello redis
hello

$ docker rm -v hello
```

In this example, the volume for `/foo` remains intact, but the volume for
`/bar` is removed. The same behavior holds for volumes inherited with
`--volumes-from`.
