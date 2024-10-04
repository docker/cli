# stop

<!---MARKER_GEN_START-->
Stop one or more running containers

### Aliases

`docker container stop`, `docker stop`

### Options

| Name                                   | Type     | Default | Description                                  |
|:---------------------------------------|:---------|:--------|:---------------------------------------------|
| [`-s`](#signal), [`--signal`](#signal) | `string` |         | Signal to send to the container              |
| [`-t`](#time), [`--time`](#time)       | `int`    | `0`     | Seconds to wait before killing the container |


<!---MARKER_GEN_END-->

## Description

The main process inside the container will receive `SIGTERM`, and after a grace
period, `SIGKILL`. The first signal can be changed with the `STOPSIGNAL`
instruction in the container's Dockerfile, or the `--stop-signal` option to
`docker run` and `docker create`.

## Examples

```console
$ docker stop my_container
```

### <a name="signal"></a> Stop container with signal (-s, --signal)

The `--signal` flag sends the system call signal to the container to exit.
This signal can be a signal name in the format `SIG<NAME>`, for instance
`SIGKILL`, or an unsigned number that matches a position in the kernel's
syscall table, for instance `9`. Refer to [signal(7)](https://man7.org/linux/man-pages/man7/signal.7.html)
for available signals.

The default signal to use is defined by the image's [`StopSignal`](https://github.com/opencontainers/image-spec/blob/v1.1.0/config.md),
which can be set through the [`STOPSIGNAL`](https://docs.docker.com/reference/dockerfile/#stopsignal)
Dockerfile instruction when building the image, or configured using the
[`--stop-signal`](https://docs.docker.com/reference/cli/docker/container/run/#stop-signal)
option when creating the container. If no signal is configured for the
container, `SIGTERM` is used as default.

### <a name="time"></a> Stop container with timeout (-t, --timeout)

The `--time` flag sets the number of seconds to wait for the container
to stop after sending the pre-defined (see [`--signal`]{#signal)) system call signal.
If the container does not exit after the timeout elapses, it's forcibly killed
with a `SIGKILL` signal.

If you set `--time` to `-1`, no timeout is applied, and the daemon
waits indefinitely for the container to exit.

The default timeout can be specified using the [`--stop-timeout`](https://docs.docker.com/reference/cli/docker/container/run/#stop-timeout)
option when creating the container. If no default is configured for the container,
the Daemon determines the default, and is 10 seconds for Linux containers, and
30 seconds for Windows containers.
