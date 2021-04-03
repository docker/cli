---
title: "stop"
description: "The stop command description and usage"
keywords: "stop, SIGKILL, SIGTERM"
---

# stop

```markdown
Usage:  docker stop [OPTIONS] CONTAINER [CONTAINER...]

Stop one or more running containers

Options:
      --help       Print usage
  -t, --time int   Seconds to wait for stop before killing it (default 10)
```

## Description

The main process inside the container will receive `SIGTERM`, and after a grace
period, `SIGKILL`. The first signal can be changed with the `STOPSIGNAL`
instruction in the container's Dockerfile, or the `--stop-signal` option to
`docker run`.

## Examples

```bash
$ docker stop my_container
```
