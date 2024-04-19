# stop

<!---MARKER_GEN_START-->
Stop one or more running containers

### Aliases

`docker container stop`, `docker stop`

### Options

| Name             | Type     | Default | Description                                  |
|:-----------------|:---------|:--------|:---------------------------------------------|
| `-s`, `--signal` | `string` |         | Signal to send to the container              |
| `-t`, `--time`   | `int`    | `0`     | Seconds to wait before killing the container |


<!---MARKER_GEN_END-->

## Description

The main process inside the container will receive `SIGTERM`, and after a grace
period, `SIGKILL`. The first signal can be changed with the `STOPSIGNAL`
instruction in the container's Dockerfile, or the `--stop-signal` option to
`docker run`.

## Examples

```console
$ docker stop my_container
```
