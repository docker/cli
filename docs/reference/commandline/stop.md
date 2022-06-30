---
title: "stop"
description: "The stop command description and usage"
keywords: "stop, SIGKILL, SIGTERM"
---

# stop

```markdown
Usage:  docker stop [OPTIONS] CONTAINER [CONTAINER...]

Stop one or more running containers

Aliases:
  docker container stop, docker stop

Options:
  -s, --signal string   Signal to send to the container
  -t, --time int        Seconds to wait before killing the container
```

## Description

The main process inside the container will receive `SIGTERM`, and after a grace
period, `SIGKILL`. The first signal can be changed with the `STOPSIGNAL`
instruction in the container's Dockerfile, or the `--stop-signal` option to
`docker run`.

## Examples

```console
$ docker stop my_container
```
