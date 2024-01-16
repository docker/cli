# docker update

<!---MARKER_GEN_START-->
Update configuration of one or more containers

### Aliases

`docker container update`, `docker update`

### Options

| Name                   | Type      | Default | Description                                                                  |
|:-----------------------|:----------|:--------|:-----------------------------------------------------------------------------|
| `--blkio-weight`       | `uint16`  | `0`     | Block IO (relative weight), between 10 and 1000, or 0 to disable (default 0) |
| `--cpu-period`         | `int64`   | `0`     | Limit CPU CFS (Completely Fair Scheduler) period                             |
| `--cpu-quota`          | `int64`   | `0`     | Limit CPU CFS (Completely Fair Scheduler) quota                              |
| `--cpu-rt-period`      | `int64`   | `0`     | Limit the CPU real-time period in microseconds                               |
| `--cpu-rt-runtime`     | `int64`   | `0`     | Limit the CPU real-time runtime in microseconds                              |
| `-c`, `--cpu-shares`   | `int64`   | `0`     | CPU shares (relative weight)                                                 |
| `--cpus`               | `decimal` |         | Number of CPUs                                                               |
| `--cpuset-cpus`        | `string`  |         | CPUs in which to allow execution (0-3, 0,1)                                  |
| `--cpuset-mems`        | `string`  |         | MEMs in which to allow execution (0-3, 0,1)                                  |
| `-m`, `--memory`       | `bytes`   | `0`     | Memory limit                                                                 |
| `--memory-reservation` | `bytes`   | `0`     | Memory soft limit                                                            |
| `--memory-swap`        | `bytes`   | `0`     | Swap limit equal to memory plus swap: -1 to enable unlimited swap            |
| `--pids-limit`         | `int64`   | `0`     | Tune container pids limit (set -1 for unlimited)                             |
| `--restart`            | `string`  |         | Restart policy to apply when a container exits                               |


<!---MARKER_GEN_END-->

