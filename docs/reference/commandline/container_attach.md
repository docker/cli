# attach

<!---MARKER_GEN_START-->
Attach local standard input, output, and error streams to a running container

### Aliases

`docker container attach`, `docker attach`

### Options

| Name                            | Type     | Default | Description                                         |
|:--------------------------------|:---------|:--------|:----------------------------------------------------|
| [`--detach-keys`](#detach-keys) | `string` |         | Override the key sequence for detaching a container |
| `--no-stdin`                    | `bool`   |         | Do not attach STDIN                                 |
| `--sig-proxy`                   | `bool`   | `true`  | Proxy all received signals to the process           |


<!---MARKER_GEN_END-->

## Description

Use `docker attach` to attach your terminal's standard input, output, and error
(or any combination of the three) to a running container using the container's
ID or name. This lets you view its output or control it interactively, as
though the commands were running directly in your terminal.

> [!NOTE]
> The `attach` command displays the output of the container's `ENTRYPOINT` and
> `CMD` process. This can appear as if the attach command is hung when in fact
> the process may simply not be writing any output at that time.

You can attach to the same contained process multiple times simultaneously,
from different sessions on the Docker host.

To stop a container, use `CTRL-c`. This key sequence sends `SIGKILL` to the
container. If `--sig-proxy` is true (the default),`CTRL-c` sends a `SIGINT` to
the container. If the container was run with `-i` and `-t`, you can detach from
a container and leave it running using the `CTRL-p CTRL-q` key sequence.

> [!NOTE]
> A process running as PID 1 inside a container is treated specially by
> Linux: it ignores any signal with the default action. So, the process
> doesn't terminate on `SIGINT` or `SIGTERM` unless it's coded to do so.

You can't redirect the standard input of a `docker attach` command while
attaching to a TTY-enabled container (using the `-i` and `-t` options).

While a client is connected to container's `stdio` using `docker attach`,
Docker uses a ~1MB memory buffer to maximize the throughput of the application.
Once this buffer is full, the speed of the API connection is affected, and so
this impacts the output process' writing speed. This is similar to other
applications like SSH. Because of this, it isn't recommended to run
performance-critical applications that generate a lot of output in the
foreground over a slow client connection. Instead, use the `docker logs`
command to get access to the logs.

## Examples

### Attach to and detach from a running container

The following example starts an Alpine container running `top` in detached mode,
then attaches to the container;

```console
$ docker run -d --name topdemo alpine top -b

$ docker attach topdemo

Mem: 2395856K used, 5638884K free, 2328K shrd, 61904K buff, 1524264K cached
CPU:   0% usr   0% sys   0% nic  99% idle   0% io   0% irq   0% sirq
Load average: 0.15 0.06 0.01 1/567 6
  PID  PPID USER     STAT   VSZ %VSZ CPU %CPU COMMAND
    1     0 root     R     1700   0%   3   0% top -b
```

As the container was started without the `-i`, and `-t` options, signals are
forwarded to the attached process, which means that the default `CTRL-p CTRL-q`
detach key sequence produces no effect, but pressing `CTRL-c` terminates the
container:

```console
<...>
  PID  PPID USER     STAT   VSZ %VSZ CPU %CPU COMMAND
    1     0 root     R     1700   0%   7   0% top -b
^P^Q
^C

$ docker ps -a --filter name=topdemo

CONTAINER ID   IMAGE     COMMAND    CREATED          STATUS                       PORTS     NAMES
96254a235bd6   alpine    "top -b"   44 seconds ago   Exited (130) 8 seconds ago             topdemo
```

Repeating the example above, but this time with the `-i` and `-t` options set;

```console
$ docker run -dit --name topdemo2 alpine /usr/bin/top -b
```

Now, when attaching to the container, and pressing the `CTRL-p CTRL-q` ("read
escape sequence"), the Docker CLI is handling the detach sequence, and the
`attach` command is detached from the container. Checking the container's status
with `docker ps` shows that the container is still running in the background:

```console
$ docker attach topdemo2

Mem: 2405344K used, 5629396K free, 2512K shrd, 65100K buff, 1524952K cached
CPU:   0% usr   0% sys   0% nic  99% idle   0% io   0% irq   0% sirq
Load average: 0.12 0.12 0.05 1/594 6
  PID  PPID USER     STAT   VSZ %VSZ CPU %CPU COMMAND
    1     0 root     R     1700   0%   3   0% top -b
read escape sequence

$ docker ps -a --filter name=topdemo2

CONTAINER ID   IMAGE     COMMAND    CREATED          STATUS          PORTS     NAMES
fde88b83c2c2   alpine    "top -b"   22 seconds ago   Up 21 seconds             topdemo2
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

### <a name="detach-keys"></a> Override the detach sequence (--detach-keys)

Use the `--detach-keys` option to override the Docker key sequence for detach.
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
containers, see [**Configuration file** section](https://docs.docker.com/reference/cli/docker/#configuration-files).
