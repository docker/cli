---
title: "attach"
description: "The attach command description and usage"
keywords: "attach, running, container"
---

# attach

```markdown
Usage: docker attach [OPTIONS] CONTAINER

Attach local standard input, output, and error streams to a running container

Options:
      --detach-keys string   Override the key sequence for detaching a container
      --help                 Print usage
      --no-stdin             Do not attach STDIN
      --sig-proxy            Proxy all received signals to the process (default true)
```

## Description

Use `docker attach` to attach your terminal's standard input, output, and error
(or any combination of the three) to a running container using the container's
ID or name. This allows you to view its ongoing output or to control it
interactively, as though the commands were running directly in your terminal.

> **Note:**
> The `attach` command will display the output of the `ENTRYPOINT/CMD` process.  This
> can appear as if the attach command is hung when in fact the process may simply
> not be interacting with the terminal at that time.

You can attach to the same contained process multiple times simultaneously,
from different sessions on the Docker host.

To stop a container, use `CTRL-c`. This key sequence sends `SIGKILL` to the
container. If `--sig-proxy` is true (the default),`CTRL-c` sends a `SIGINT` to
the container. If the container was run with `-i` and `-t`, you can detach from
a container and leave it running using the `CTRL-p CTRL-q` key sequence.

> **Note:**
> A process running as PID 1 inside a container is treated specially by
> Linux: it ignores any signal with the default action. So, the process
> will not terminate on `SIGINT` or `SIGTERM` unless it is coded to do
> so.

It is forbidden to redirect the standard input of a `docker attach` command
while attaching to a TTY-enabled container (using the `-i` and `-t` options).

While a client is connected to container's `stdio` using `docker attach`, Docker
uses a ~1MB memory buffer to maximize the throughput of the application.
Once this buffer is full, the speed of the API connection is affected, and so
this impacts the output process' writing speed. This is similar to other
applications like SSH. Because of this, it is not recommended to run
performance critical applications that generate a lot of output in the
foreground over a slow client connection. Instead, users should use the
`docker logs` command to get access to the logs.

### Override the detach sequence

If you want, you can configure an override the Docker key sequence for detach.
This is useful if the Docker default sequence conflicts with key sequence you
use for other applications. There are two ways to define your own detach key
sequence, as a per-container override or as a configuration property on  your
entire configuration.

To override the sequence for an individual container, use the
`--detach-keys="<sequence>"` flag with the `docker attach` command. The format of
the `<sequence>` is either a letter [a-Z], or the `ctrl-` combined with any of
the following:

* `a-z` (a single lowercase alpha character )
* `@` (at sign)
* `[` (left bracket)
* `\\` (two backward slashes)
*  `_` (underscore)
* `^` (caret)

These `a`, `ctrl-a`, `X`, or `ctrl-\\` values are all examples of valid key
sequences. To configure a different configuration default key sequence for all
containers, see [**Configuration file** section](cli.md#configuration-files).

## Examples

### Attach to and detach from a running container

The following example starts an ubuntu container running `top` in detached mode,
then attaches to the container;

```console
$ docker run -d --name topdemo ubuntu:22.04 /usr/bin/top -b

$ docker attach topdemo

top - 12:27:44 up 3 days, 21:54,  0 users,  load average: 0.00, 0.00, 0.00
Tasks:   1 total,   1 running,   0 sleeping,   0 stopped,   0 zombie
%Cpu(s):  0.1 us,  0.1 sy,  0.0 ni, 99.8 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
MiB Mem :   3934.3 total,    770.1 free,    674.2 used,   2490.1 buff/cache
MiB Swap:   1024.0 total,    839.3 free,    184.7 used.   2814.0 avail Mem

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
    1 root      20   0    7180   2896   2568 R   0.0   0.1   0:00.02 top
```

As the container was started without the `-i`, and `-t` options, signals are
forwarded to the attached process, which means that the default `CTRL-p CTRL-q`
detach key sequence produces no effect, but pressing `CTRL-c` terminates the
container:

```console
<...>
  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
    1 root      20   0    7180   2896   2568 R   0.0   0.1   0:00.02 top^P^Q
^C

$ docker ps -a --filter name=topdemo

CONTAINER ID   IMAGE          COMMAND             CREATED              STATUS                          PORTS     NAMES
4cf0d0ebb079   ubuntu:22.04   "/usr/bin/top -b"   About a minute ago   Exited (0) About a minute ago             topdemo
```

Repeating the example above, but this time with the `-i` and `-t` options set;

```console
$ docker run -dit --name topdemo2 ubuntu:22.04 /usr/bin/top -b
```

Now, when attaching to the container, and pressing the `CTRL-p CTRL-q` ("read
escape sequence"), the Docker CLI is handling the detach sequence, and the
`attach` command is detached from the container. Checking the container's status
with `docker ps` shows that the container is still running in the background:

```console
$ docker attach topdemo2

top - 12:44:32 up 3 days, 22:11,  0 users,  load average: 0.00, 0.00, 0.00
Tasks:   1 total,   1 running,   0 sleeping,   0 stopped,   0 zombie
%Cpu(s): 50.0 us,  0.0 sy,  0.0 ni, 50.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
MiB Mem :   3934.3 total,    770.6 free,    672.4 used,   2491.4 buff/cache
MiB Swap:   1024.0 total,    839.3 free,    184.7 used.   2815.8 avail Mem

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
    1 root      20   0    7180   2776   2452 R   0.0   0.1   0:00.02 topread escape sequence

$ docker ps -a --filter name=topdemo2

CONTAINER ID   IMAGE          COMMAND             CREATED         STATUS         PORTS     NAMES
b1661dce0fc2   ubuntu:22.04   "/usr/bin/top -b"   2 minutes ago   Up 2 minutes             topdemo2
```

### Get the exit code of the container's command

And in this second example, you can see the exit code returned by the `bash`
process is returned by the `docker attach` command to its caller too:

```console
$ docker run --name test -dit alpine
275c44472aebd77c926d4527885bb09f2f6db21d878c75f0a1c212c03d3bcfab

$ docker attach test
/# exit 13

$ echo $?
13

$ docker ps -a --filter name=test

CONTAINER ID   IMAGE     COMMAND     CREATED              STATUS                       PORTS     NAMES
a2fe3fd886db   alpine    "/bin/sh"   About a minute ago   Exited (13) 40 seconds ago             test
```
