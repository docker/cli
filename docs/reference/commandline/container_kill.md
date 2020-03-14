---
title: "kill"
description: "The kill command description and usage"
keywords: "container, kill, signal"
---

# kill

```markdown
Usage:  docker kill [OPTIONS] CONTAINER [CONTAINER...]

Kill one or more running containers

Options:
      --help            Print usage
  -s, --signal string   Signal to send to the container (default "KILL")
```

## Description

The `docker kill` subcommand kills one or more containers. The main process
inside the container is sent `SIGKILL` signal (default), or the signal that is
specified with the `--signal` option. You can kill a container using the
container's ID, ID-prefix, or name.

> **Note**
>
> `ENTRYPOINT` and `CMD` in the *shell* form run as a child process of
> `/bin/sh -c`, which does not pass signals. This means that the executable is
> not the container’s PID 1 and does not receive Unix signals.

## Examples


### Send a KILL signal  to a container

The following example sends the default `KILL` signal to the container named
`my_container`:

```bash
$ docker kill my_container
```

### Send a custom signal  to a container

The following example sends a `SIGHUP` signal to the container named
`my_container`:

```bash
$ docker kill --signal=SIGHUP  my_container
```


You can specify a custom signal either by _name_, or _number_. The `SIG` prefix
is optional, so the following examples are equivalent:

```bash
$ docker kill --signal=SIGHUP my_container
$ docker kill --signal=HUP my_container
$ docker kill --signal=1 my_container
```

Refer to the [`signal(7)`](http://man7.org/linux/man-pages/man7/signal.7.html)
man-page for a list of standard Linux signals.
