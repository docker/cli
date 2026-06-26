## update

<!---MARKER_GEN_START-->
Update configuration of one or more containers

### Aliases

`docker container update`, `docker update`

### Options

| Name                                               | Type      | Default | Description                                                                  |
|:---------------------------------------------------|:----------|:--------|:-----------------------------------------------------------------------------|
| `--blkio-weight`                                   | `uint16`  | `0`     | Block IO (relative weight), between 10 and 1000, or 0 to disable (default 0) |
| `--cpu-period`                                     | `int64`   | `0`     | Limit CPU CFS (Completely Fair Scheduler) period                             |
| `--cpu-quota`                                      | `int64`   | `0`     | Limit CPU CFS (Completely Fair Scheduler) quota                              |
| `--cpu-rt-period`                                  | `int64`   | `0`     | Limit the CPU real-time period in microseconds                               |
| `--cpu-rt-runtime`                                 | `int64`   | `0`     | Limit the CPU real-time runtime in microseconds                              |
| [`-c`](#cpu-shares), [`--cpu-shares`](#cpu-shares) | `int64`   | `0`     | CPU shares (relative weight)                                                 |
| `--cpus`                                           | `decimal` |         | Number of CPUs                                                               |
| `--cpuset-cpus`                                    | `string`  |         | CPUs in which to allow execution (0-3, 0,1)                                  |
| `--cpuset-mems`                                    | `string`  |         | MEMs in which to allow execution (0-3, 0,1)                                  |
| [`-m`](#memory), [`--memory`](#memory)             | `bytes`   | `0`     | Memory limit                                                                 |
| `--memory-reservation`                             | `bytes`   | `0`     | Memory soft limit                                                            |
| `--memory-swap`                                    | `bytes`   | `0`     | Swap limit equal to memory plus swap: -1 to enable unlimited swap            |
| `--pids-limit`                                     | `int64`   | `0`     | Tune container pids limit (set -1 for unlimited)                             |
| [`--restart`](#restart)                            | `string`  |         | Restart policy to apply when a container exits                               |


<!---MARKER_GEN_END-->

## Description

The `docker update` command dynamically updates container configuration.
You can use this command to prevent containers from consuming too many
resources from their Docker host.  With a single command, you can place
limits on a single container or on many. To specify more than one container,
provide space-separated list of container names or IDs.

> [!WARNING]
> The `docker update` and `docker container update` commands are not supported
> for Windows containers.
{ .warning }

## Examples

The following sections illustrate ways to use this command.

### <a name="cpu-shares"></a> Update a container's cpu-shares (--cpu-shares)

To limit a container's cpu-shares to 512, first identify the container
name or ID. You can use `docker ps` to find these values. You can also
use the ID returned from the `docker run` command.  Then, do the following:

```console
$ docker update --cpu-shares 512 abebf7571666
```

### <a name="memory"></a> Update a container with cpu-shares and memory (-m, --memory)

To update multiple resource configurations for multiple containers:

```console
$ docker update --cpu-shares 512 -m 300M abebf7571666 hopeful_morse
```

### <a name="restart"></a> Update a container's restart policy (--restart)

You can change a container's restart policy on a running container. The new
restart policy takes effect instantly after you run `docker update` on a
container.

To update restart policy for one or more containers:

```console
$ docker update --restart=on-failure:3 abebf7571666 hopeful_morse
```

Note that if the container is started with `--rm` flag, you cannot update the restart
policy for it. The `AutoRemove` and `RestartPolicy` are mutually exclusive for the
container.
