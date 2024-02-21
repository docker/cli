# docker build

<!---MARKER_GEN_START-->
Build an image from a Dockerfile

### Aliases

`docker image build`, `docker build`, `docker buildx build`, `docker builder build`

### Options

| Name                      | Type          | Default   | Description                                                       |
|:--------------------------|:--------------|:----------|:------------------------------------------------------------------|
| `--add-host`              | `list`        |           | Add a custom host-to-IP mapping (`host:ip`)                       |
| `--build-arg`             | `list`        |           | Set build-time variables                                          |
| `--cache-from`            | `stringSlice` |           | Images to consider as cache sources                               |
| `--cgroup-parent`         | `string`      |           | Set the parent cgroup for the `RUN` instructions during build     |
| `--compress`              |               |           | Compress the build context using gzip                             |
| `--cpu-period`            | `int64`       | `0`       | Limit the CPU CFS (Completely Fair Scheduler) period              |
| `--cpu-quota`             | `int64`       | `0`       | Limit the CPU CFS (Completely Fair Scheduler) quota               |
| `-c`, `--cpu-shares`      | `int64`       | `0`       | CPU shares (relative weight)                                      |
| `--cpuset-cpus`           | `string`      |           | CPUs in which to allow execution (0-3, 0,1)                       |
| `--cpuset-mems`           | `string`      |           | MEMs in which to allow execution (0-3, 0,1)                       |
| `--disable-content-trust` | `bool`        | `true`    | Skip image verification                                           |
| `-f`, `--file`            | `string`      |           | Name of the Dockerfile (Default is `PATH/Dockerfile`)             |
| `--force-rm`              |               |           | Always remove intermediate containers                             |
| `--iidfile`               | `string`      |           | Write the image ID to the file                                    |
| `--isolation`             | `string`      |           | Container isolation technology                                    |
| `--label`                 | `list`        |           | Set metadata for an image                                         |
| `-m`, `--memory`          | `bytes`       | `0`       | Memory limit                                                      |
| `--memory-swap`           | `bytes`       | `0`       | Swap limit equal to memory plus swap: -1 to enable unlimited swap |
| `--network`               | `string`      | `default` | Set the networking mode for the RUN instructions during build     |
| `--no-cache`              |               |           | Do not use cache when building the image                          |
| `--platform`              | `string`      |           | Set platform if server is multi-platform capable                  |
| `--pull`                  |               |           | Always attempt to pull a newer version of the image               |
| `-q`, `--quiet`           |               |           | Suppress the build output and print image ID on success           |
| `--rm`                    | `bool`        | `true`    | Remove intermediate containers after a successful build           |
| `--security-opt`          | `stringSlice` |           | Security options                                                  |
| `--shm-size`              | `bytes`       | `0`       | Size of `/dev/shm`                                                |
| `--squash`                |               |           | Squash newly built layers into a single new layer                 |
| `-t`, `--tag`             | `list`        |           | Name and optionally a tag in the `name:tag` format                |
| `--target`                | `string`      |           | Set the target build stage to build.                              |
| `--ulimit`                | `ulimit`      |           | Ulimit options                                                    |


<!---MARKER_GEN_END-->

