# start

<!---MARKER_GEN_START-->
Start one or more stopped containers

### Aliases

`docker container start`, `docker start`

### Options

| Name                  | Type     | Default | Description                                         |
|:----------------------|:---------|:--------|:----------------------------------------------------|
| `-a`, `--attach`      | `bool`   |         | Attach STDOUT/STDERR and forward signals            |
| `--checkpoint`        | `string` |         | Restore from this checkpoint                        |
| `--checkpoint-dir`    | `string` |         | Use a custom checkpoint storage directory           |
| `--detach-keys`       | `string` |         | Override the key sequence for detaching a container |
| `-i`, `--interactive` | `bool`   |         | Attach container's STDIN                            |


<!---MARKER_GEN_END-->

## Description

By default `docker start` starts each container in the background and
prints its name. It does not attach your terminal.

`--attach` (`-a`) attaches STDOUT/STDERR and forwards signals. You can
only attach to one container. Combine `-a` with `-i` to attach STDIN as
well (the container must have been created with `-i` for STDIN to be
open).

`--attach` is not the same as [`docker attach`](container_attach.md):
`start -a` starts a stopped container and then attaches; `attach`
connects to a container that is already running.

## Examples

```console
$ docker start my_container
```

```console
$ docker start -ai my_container
```
