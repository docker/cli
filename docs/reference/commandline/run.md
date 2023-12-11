# run

<!---MARKER_GEN_START-->
Create and run a new container from an image

### Aliases

`docker container run`, `docker run`

### Options

| Name                                          | Type          | Default   | Description                                                                                                                                                                                                                                                                                                      |
|:----------------------------------------------|:--------------|:----------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`--add-host`](#add-host)                     | `list`        |           | Add a custom host-to-IP mapping (host:ip)                                                                                                                                                                                                                                                                        |
| `--annotation`                                | `map`         | `map[]`   | Add an annotation to the container (passed through to the OCI runtime)                                                                                                                                                                                                                                           |
| [`-a`](#attach), [`--attach`](#attach)        | `list`        |           | Attach to STDIN, STDOUT or STDERR                                                                                                                                                                                                                                                                                |
| `--blkio-weight`                              | `uint16`      | `0`       | Block IO (relative weight), between 10 and 1000, or 0 to disable (default 0)                                                                                                                                                                                                                                     |
| `--blkio-weight-device`                       | `list`        |           | Block IO weight (relative device weight)                                                                                                                                                                                                                                                                         |
| `--cap-add`                                   | `list`        |           | Add Linux capabilities                                                                                                                                                                                                                                                                                           |
| `--cap-drop`                                  | `list`        |           | Drop Linux capabilities                                                                                                                                                                                                                                                                                          |
| `--cgroup-parent`                             | `string`      |           | Optional parent cgroup for the container                                                                                                                                                                                                                                                                         |
| `--cgroupns`                                  | `string`      |           | Cgroup namespace to use (host\|private)<br>'host':    Run the container in the Docker host's cgroup namespace<br>'private': Run the container in its own private cgroup namespace<br>'':        Use the cgroup namespace as configured by the<br>           default-cgroupns-mode option on the daemon (default) |
| [`--cidfile`](#cidfile)                       | `string`      |           | Write the container ID to the file                                                                                                                                                                                                                                                                               |
| `--cpu-count`                                 | `int64`       | `0`       | CPU count (Windows only)                                                                                                                                                                                                                                                                                         |
| `--cpu-percent`                               | `int64`       | `0`       | CPU percent (Windows only)                                                                                                                                                                                                                                                                                       |
| `--cpu-period`                                | `int64`       | `0`       | Limit CPU CFS (Completely Fair Scheduler) period                                                                                                                                                                                                                                                                 |
| `--cpu-quota`                                 | `int64`       | `0`       | Limit CPU CFS (Completely Fair Scheduler) quota                                                                                                                                                                                                                                                                  |
| `--cpu-rt-period`                             | `int64`       | `0`       | Limit CPU real-time period in microseconds                                                                                                                                                                                                                                                                       |
| `--cpu-rt-runtime`                            | `int64`       | `0`       | Limit CPU real-time runtime in microseconds                                                                                                                                                                                                                                                                      |
| `-c`, `--cpu-shares`                          | `int64`       | `0`       | CPU shares (relative weight)                                                                                                                                                                                                                                                                                     |
| `--cpus`                                      | `decimal`     |           | Number of CPUs                                                                                                                                                                                                                                                                                                   |
| `--cpuset-cpus`                               | `string`      |           | CPUs in which to allow execution (0-3, 0,1)                                                                                                                                                                                                                                                                      |
| `--cpuset-mems`                               | `string`      |           | MEMs in which to allow execution (0-3, 0,1)                                                                                                                                                                                                                                                                      |
| `-d`, `--detach`                              |               |           | Run container in background and print container ID                                                                                                                                                                                                                                                               |
| [`--detach-keys`](#detach-keys)               | `string`      |           | Override the key sequence for detaching a container                                                                                                                                                                                                                                                              |
| [`--device`](#device)                         | `list`        |           | Add a host device to the container                                                                                                                                                                                                                                                                               |
| [`--device-cgroup-rule`](#device-cgroup-rule) | `list`        |           | Add a rule to the cgroup allowed devices list                                                                                                                                                                                                                                                                    |
| `--device-read-bps`                           | `list`        |           | Limit read rate (bytes per second) from a device                                                                                                                                                                                                                                                                 |
| `--device-read-iops`                          | `list`        |           | Limit read rate (IO per second) from a device                                                                                                                                                                                                                                                                    |
| `--device-write-bps`                          | `list`        |           | Limit write rate (bytes per second) to a device                                                                                                                                                                                                                                                                  |
| `--device-write-iops`                         | `list`        |           | Limit write rate (IO per second) to a device                                                                                                                                                                                                                                                                     |
| `--disable-content-trust`                     |               |           | Skip image verification                                                                                                                                                                                                                                                                                          |
| `--dns`                                       | `list`        |           | Set custom DNS servers                                                                                                                                                                                                                                                                                           |
| `--dns-option`                                | `list`        |           | Set DNS options                                                                                                                                                                                                                                                                                                  |
| `--dns-search`                                | `list`        |           | Set custom DNS search domains                                                                                                                                                                                                                                                                                    |
| `--domainname`                                | `string`      |           | Container NIS domain name                                                                                                                                                                                                                                                                                        |
| `--entrypoint`                                | `string`      |           | Overwrite the default ENTRYPOINT of the image                                                                                                                                                                                                                                                                    |
| [`-e`](#env), [`--env`](#env)                 | `list`        |           | Set environment variables                                                                                                                                                                                                                                                                                        |
| `--env-file`                                  | `list`        |           | Read in a file of environment variables                                                                                                                                                                                                                                                                          |
| `--expose`                                    | `list`        |           | Expose a port or a range of ports                                                                                                                                                                                                                                                                                |
| [`--gpus`](#gpus)                             | `gpu-request` |           | GPU devices to add to the container ('all' to pass all GPUs)                                                                                                                                                                                                                                                     |
| `--group-add`                                 | `list`        |           | Add additional groups to join                                                                                                                                                                                                                                                                                    |
| `--health-cmd`                                | `string`      |           | Command to run to check health                                                                                                                                                                                                                                                                                   |
| `--health-interval`                           | `duration`    | `0s`      | Time between running the check (ms\|s\|m\|h) (default 0s)                                                                                                                                                                                                                                                        |
| `--health-retries`                            | `int`         | `0`       | Consecutive failures needed to report unhealthy                                                                                                                                                                                                                                                                  |
| `--health-start-period`                       | `duration`    | `0s`      | Start period for the container to initialize before starting health-retries countdown (ms\|s\|m\|h) (default 0s)                                                                                                                                                                                                 |
| `--health-timeout`                            | `duration`    | `0s`      | Maximum time to allow one check to run (ms\|s\|m\|h) (default 0s)                                                                                                                                                                                                                                                |
| `--help`                                      |               |           | Print usage                                                                                                                                                                                                                                                                                                      |
| `-h`, `--hostname`                            | `string`      |           | Container host name                                                                                                                                                                                                                                                                                              |
| `--init`                                      |               |           | Run an init inside the container that forwards signals and reaps processes                                                                                                                                                                                                                                       |
| `-i`, `--interactive`                         |               |           | Keep STDIN open even if not attached                                                                                                                                                                                                                                                                             |
| `--io-maxbandwidth`                           | `bytes`       | `0`       | Maximum IO bandwidth limit for the system drive (Windows only)                                                                                                                                                                                                                                                   |
| `--io-maxiops`                                | `uint64`      | `0`       | Maximum IOps limit for the system drive (Windows only)                                                                                                                                                                                                                                                           |
| `--ip`                                        | `string`      |           | IPv4 address (e.g., 172.30.100.104)                                                                                                                                                                                                                                                                              |
| `--ip6`                                       | `string`      |           | IPv6 address (e.g., 2001:db8::33)                                                                                                                                                                                                                                                                                |
| `--ipc`                                       | `string`      |           | IPC mode to use                                                                                                                                                                                                                                                                                                  |
| [`--isolation`](#isolation)                   | `string`      |           | Container isolation technology                                                                                                                                                                                                                                                                                   |
| `--kernel-memory`                             | `bytes`       | `0`       | Kernel memory limit                                                                                                                                                                                                                                                                                              |
| [`-l`](#label), [`--label`](#label)           | `list`        |           | Set meta data on a container                                                                                                                                                                                                                                                                                     |
| `--label-file`                                | `list`        |           | Read in a line delimited file of labels                                                                                                                                                                                                                                                                          |
| `--link`                                      | `list`        |           | Add link to another container                                                                                                                                                                                                                                                                                    |
| `--link-local-ip`                             | `list`        |           | Container IPv4/IPv6 link-local addresses                                                                                                                                                                                                                                                                         |
| `--log-driver`                                | `string`      |           | Logging driver for the container                                                                                                                                                                                                                                                                                 |
| `--log-opt`                                   | `list`        |           | Log driver options                                                                                                                                                                                                                                                                                               |
| `--mac-address`                               | `string`      |           | Container MAC address (e.g., 92:d0:c6:0a:29:33)                                                                                                                                                                                                                                                                  |
| [`-m`](#memory), [`--memory`](#memory)        | `bytes`       | `0`       | Memory limit                                                                                                                                                                                                                                                                                                     |
| `--memory-reservation`                        | `bytes`       | `0`       | Memory soft limit                                                                                                                                                                                                                                                                                                |
| `--memory-swap`                               | `bytes`       | `0`       | Swap limit equal to memory plus swap: '-1' to enable unlimited swap                                                                                                                                                                                                                                              |
| `--memory-swappiness`                         | `int64`       | `-1`      | Tune container memory swappiness (0 to 100)                                                                                                                                                                                                                                                                      |
| [`--mount`](#mount)                           | `mount`       |           | Attach a filesystem mount to the container                                                                                                                                                                                                                                                                       |
| [`--name`](#name)                             | `string`      |           | Assign a name to the container                                                                                                                                                                                                                                                                                   |
| [`--network`](#network)                       | `network`     |           | Connect a container to a network                                                                                                                                                                                                                                                                                 |
| `--network-alias`                             | `list`        |           | Add network-scoped alias for the container                                                                                                                                                                                                                                                                       |
| `--no-healthcheck`                            |               |           | Disable any container-specified HEALTHCHECK                                                                                                                                                                                                                                                                      |
| `--oom-kill-disable`                          |               |           | Disable OOM Killer                                                                                                                                                                                                                                                                                               |
| `--oom-score-adj`                             | `int`         | `0`       | Tune host's OOM preferences (-1000 to 1000)                                                                                                                                                                                                                                                                      |
| `--pid`                                       | `string`      |           | PID namespace to use                                                                                                                                                                                                                                                                                             |
| `--pids-limit`                                | `int64`       | `0`       | Tune container pids limit (set -1 for unlimited)                                                                                                                                                                                                                                                                 |
| `--platform`                                  | `string`      |           | Set platform if server is multi-platform capable                                                                                                                                                                                                                                                                 |
| [`--privileged`](#privileged)                 |               |           | Give extended privileges to this container                                                                                                                                                                                                                                                                       |
| [`-p`](#publish), [`--publish`](#publish)     | `list`        |           | Publish a container's port(s) to the host                                                                                                                                                                                                                                                                        |
| `-P`, `--publish-all`                         |               |           | Publish all exposed ports to random ports                                                                                                                                                                                                                                                                        |
| [`--pull`](#pull)                             | `string`      | `missing` | Pull image before running (`always`, `missing`, `never`)                                                                                                                                                                                                                                                         |
| `-q`, `--quiet`                               |               |           | Suppress the pull output                                                                                                                                                                                                                                                                                         |
| [`--read-only`](#read-only)                   |               |           | Mount the container's root filesystem as read only                                                                                                                                                                                                                                                               |
| [`--restart`](#restart)                       | `string`      | `no`      | Restart policy to apply when a container exits                                                                                                                                                                                                                                                                   |
| `--rm`                                        |               |           | Automatically remove the container when it exits                                                                                                                                                                                                                                                                 |
| `--runtime`                                   | `string`      |           | Runtime to use for this container                                                                                                                                                                                                                                                                                |
| [`--security-opt`](#security-opt)             | `list`        |           | Security Options                                                                                                                                                                                                                                                                                                 |
| `--shm-size`                                  | `bytes`       | `0`       | Size of /dev/shm                                                                                                                                                                                                                                                                                                 |
| `--sig-proxy`                                 |               |           | Proxy received signals to the process                                                                                                                                                                                                                                                                            |
| [`--stop-signal`](#stop-signal)               | `string`      |           | Signal to stop the container                                                                                                                                                                                                                                                                                     |
| [`--stop-timeout`](#stop-timeout)             | `int`         | `0`       | Timeout (in seconds) to stop a container                                                                                                                                                                                                                                                                         |
| [`--storage-opt`](#storage-opt)               | `list`        |           | Storage driver options for the container                                                                                                                                                                                                                                                                         |
| [`--sysctl`](#sysctl)                         | `map`         | `map[]`   | Sysctl options                                                                                                                                                                                                                                                                                                   |
| [`--tmpfs`](#tmpfs)                           | `list`        |           | Mount a tmpfs directory                                                                                                                                                                                                                                                                                          |
| `-t`, `--tty`                                 |               |           | Allocate a pseudo-TTY                                                                                                                                                                                                                                                                                            |
| [`--ulimit`](#ulimit)                         | `ulimit`      |           | Ulimit options                                                                                                                                                                                                                                                                                                   |
| `-u`, `--user`                                | `string`      |           | Username or UID (format: <name\|uid>[:<group\|gid>])                                                                                                                                                                                                                                                             |
| `--userns`                                    | `string`      |           | User namespace to use                                                                                                                                                                                                                                                                                            |
| `--uts`                                       | `string`      |           | UTS namespace to use                                                                                                                                                                                                                                                                                             |
| [`-v`](#volume), [`--volume`](#volume)        | `list`        |           | Bind mount a volume                                                                                                                                                                                                                                                                                              |
| `--volume-driver`                             | `string`      |           | Optional volume driver for the container                                                                                                                                                                                                                                                                         |
| [`--volumes-from`](#volumes-from)             | `list`        |           | Mount volumes from the specified container(s)                                                                                                                                                                                                                                                                    |
| [`-w`](#workdir), [`--workdir`](#workdir)     | `string`      |           | Working directory inside the container                                                                                                                                                                                                                                                                           |


<!---MARKER_GEN_END-->

## Description

The `docker run` command runs a command in a new container, pulling the image if needed and starting the container.

You can restart a stopped container with all its previous changes intact using `docker start`.
Use `docker ps -a` to view a list of all containers, including those that are stopped.


## Examples

### <a name="name"></a> Assign name and allocate pseudo-TTY (--name, -it)

```console
$ docker run --name test -it debian

root@d6c0fe130dba:/# exit 13
$ echo $?
13
$ docker ps -a | grep test
d6c0fe130dba        debian:7            "/bin/bash"         26 seconds ago      Exited (13) 17 seconds ago                         test
```

This example runs a container named `test` using the `debian:latest`
image. The `-it` instructs Docker to allocate a pseudo-TTY connected to
the container's stdin; creating an interactive `bash` shell in the container.
The example quits the `bash` shell by entering
`exit 13`, passing the exit code on to the caller of
`docker run`, and recording it in the `test` container's metadata.

### <a name="cidfile"></a> Capture container ID (--cidfile)

```console
$ docker run --cidfile /tmp/docker_test.cid ubuntu echo "test"
```

This creates a container and prints `test` to the console. The `cidfile`
flag makes Docker attempt to create a new file and write the container ID to it.
If the file exists already, Docker returns an error. Docker closes this
file when `docker run` exits.

### <a name="privileged"></a> Full container capabilities (--privileged)

```console
$ docker run -t -i --rm ubuntu bash
root@bc338942ef20:/# mount -t tmpfs none /mnt
mount: permission denied
```

This *doesn't* work, because by default, Docker drops most potentially dangerous kernel
capabilities, including `CAP_SYS_ADMIN ` (which is required to mount
filesystems). However, the `--privileged` flag allows it to run:

```console
$ docker run -t -i --privileged ubuntu bash
root@50e3f57e16e6:/# mount -t tmpfs none /mnt
root@50e3f57e16e6:/# df -h
Filesystem      Size  Used Avail Use% Mounted on
none            1.9G     0  1.9G   0% /mnt
```

The `--privileged` flag gives *all* capabilities to the container, and it also
lifts all the limitations enforced by the `device` cgroup controller. In other
words, the container can then do almost everything that the host can do. This
flag exists to allow special use-cases, like running Docker within Docker.

### <a name="workdir"></a> Set working directory (-w, --workdir)

```console
$ docker  run -w /path/to/dir/ -i -t  ubuntu pwd
```

The `-w` option runs the command executed inside the directory specified, in this example,
`/path/to/dir/`. If the path does not exist, Docker creates it inside the container.

### <a name="storage-opt"></a> Set storage driver options per container (--storage-opt)

```console
$ docker run -it --storage-opt size=120G fedora /bin/bash
```

This (size) constraints the container filesystem size to 120G at creation time.
This option is only available for the `devicemapper`, `btrfs`, `overlay2`,
`windowsfilter` and `zfs` storage drivers.

For the `overlay2` storage driver, the size option is only available if the
backing filesystem is `xfs` and mounted with the `pquota` mount option.
Under these conditions, you can pass any size less than the backing filesystem size.

For the `windowsfilter`, `devicemapper`, `btrfs`, and `zfs` storage drivers,
you cannot pass a size less than the Default BaseFS Size.


### <a name="tmpfs"></a> Mount tmpfs (--tmpfs)

```console
$ docker run -d --tmpfs /run:rw,noexec,nosuid,size=65536k my_image
```

The `--tmpfs` flag mounts an empty tmpfs into the container with the `rw`,
`noexec`, `nosuid`, `size=65536k` options.

### <a name="volume"></a> Mount volume (-v)

```console
$ docker  run  -v $(pwd):$(pwd) -w $(pwd) -i -t  ubuntu pwd
```

The example above mounts the current directory into the container at the same path
using the `-v` flag, sets it as the working directory, and then runs the `pwd` command inside the container.

As of Docker Engine version 23, you can use relative paths on the host.

```console
$ docker  run  -v ./content:/content -w /content -i -t  ubuntu pwd
```

The example above mounts the `content` directory in the current directory into the container at the
`/content` path using the `-v` flag, sets it as the working directory, and then
runs the `pwd` command inside the container.

```console
$ docker run -v /doesnt/exist:/foo -w /foo -i -t ubuntu bash
```

When the host directory of a bind-mounted volume doesn't exist, Docker
automatically creates this directory on the host for you. In the
example above, Docker creates the `/doesnt/exist`
folder before starting your container.

### <a name="read-only"></a> Mount volume read-only (--read-only)

```console
$ docker run --read-only -v /icanwrite busybox touch /icanwrite/here
```

You can use volumes in combination with the `--read-only` flag to control where
a container writes files. The `--read-only` flag mounts the container's root
filesystem as read only prohibiting writes to locations other than the
specified volumes for the container.

```console
$ docker run -t -i -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/static-docker-binary:/usr/bin/docker busybox sh
```

By bind-mounting the Docker Unix socket and statically linked Docker
binary (refer to [get the Linux binary](https://docs.docker.com/engine/install/binaries/#install-static-binaries)),
you give the container the full access to create and manipulate the host's
Docker daemon.

On Windows, you must specify the paths using Windows-style path semantics.

```powershell
PS C:\> docker run -v c:\foo:c:\dest microsoft/nanoserver cmd /s /c type c:\dest\somefile.txt
Contents of file

PS C:\> docker run -v c:\foo:d: microsoft/nanoserver cmd /s /c type d:\somefile.txt
Contents of file
```

The following examples fails when using Windows-based containers, as the
destination of a volume or bind mount inside the container must be one of:
a non-existing or empty directory; or a drive other than `C:`. Further, the source
of a bind mount must be a local directory, not a file.

```powershell
net use z: \\remotemachine\share
docker run -v z:\foo:c:\dest ...
docker run -v \\uncpath\to\directory:c:\dest ...
docker run -v c:\foo\somefile.txt:c:\dest ...
docker run -v c:\foo:c: ...
docker run -v c:\foo:c:\existing-directory-with-contents ...
```

For in-depth information about volumes, refer to [manage data in containers](https://docs.docker.com/storage/volumes/)

### <a name="mount"></a> Add bind mounts or volumes using the --mount flag

The `--mount` flag allows you to mount volumes, host-directories, and `tmpfs`
mounts in a container.

The `--mount` flag supports most options supported by the `-v` or the
`--volume` flag, but uses a different syntax. For in-depth information on the
`--mount` flag, and a comparison between `--volume` and `--mount`, refer to
[Bind mounts](https://docs.docker.com/storage/bind-mounts/).

Even though there is no plan to deprecate `--volume`, usage of `--mount` is recommended.

Examples:

```console
$ docker run --read-only --mount type=volume,target=/icanwrite busybox touch /icanwrite/here
```

```console
$ docker run -t -i --mount type=bind,src=/data,dst=/data busybox sh
```

### <a name="publish"></a> Publish or expose port (-p, --expose)

```console
$ docker run -p 127.0.0.1:80:8080/tcp ubuntu bash
```

This binds port `8080` of the container to TCP port `80` on `127.0.0.1` of the host
machine. You can also specify `udp` and `sctp` ports.
The [Docker User Guide](https://docs.docker.com/network/links/)
explains in detail how to use ports in Docker.

Note that ports which are not bound to the host (i.e., `-p 80:80` instead of
`-p 127.0.0.1:80:80`) are externally accessible. This also applies if
you configured UFW to block this specific port, as Docker manages its
own iptables rules. [Read more](https://docs.docker.com/network/iptables/)

```console
$ docker run --expose 80 ubuntu bash
```

This exposes port `80` of the container without publishing the port to the host
system's interfaces.

### <a name="pull"></a> Set the pull policy (--pull)

Use the `--pull` flag to set the image pull policy when creating (and running)
the container.

The `--pull` flag can take one of these values:

| Value               | Description                                                                                                       |
|:--------------------|:------------------------------------------------------------------------------------------------------------------|
| `missing` (default) | Pull the image if it was not found in the image cache, or use the cached image otherwise.                         |
| `never`             | Do not pull the image, even if it's missing, and produce an error if the image does not exist in the image cache. |
| `always`            | Always perform a pull before creating the container.                                                              |

When creating (and running) a container from an image, the daemon checks if the
image exists in the local image cache. If the image is missing, an error is
returned to the CLI, allowing it to initiate a pull.

The default (`missing`) is to only pull the image if it's not present in the
daemon's image cache. This default allows you to run images that only exist
locally (for example, images you built from a Dockerfile, but that have not
been pushed to a registry), and reduces networking.

The `always` option always initiates a pull before creating the container. This
option makes sure the image is up-to-date, and prevents you from using outdated
images, but may not be suitable in situations where you want to test a locally
built image before pushing (as pulling the image overwrites the existing image
in the image cache).

The `never` option disables (implicit) pulling images when creating containers,
and only uses images that are available in the image cache. If the specified
image is not found, an error is produced, and the container is not created.
This option is useful in situations where networking is not available, or to
prevent images from being pulled implicitly when creating containers.

The following example shows `docker run` with the `--pull=never` option set,
which produces en error as the image is missing in the image-cache:

```console
$ docker run --pull=never hello-world
docker: Error response from daemon: No such image: hello-world:latest.
```

### <a name="env"></a> Set environment variables (-e, --env, --env-file)

```console
$ docker run -e MYVAR1 --env MYVAR2=foo --env-file ./env.list ubuntu bash
```

Use the `-e`, `--env`, and `--env-file` flags to set simple (non-array)
environment variables in the container you're running, or overwrite variables
defined in the Dockerfile of the image you're running.

You can define the variable and its value when running the container:

```console
$ docker run --env VAR1=value1 --env VAR2=value2 ubuntu env | grep VAR
VAR1=value1
VAR2=value2
```

You can also use variables exported to your local environment:

```console
export VAR1=value1
export VAR2=value2

$ docker run --env VAR1 --env VAR2 ubuntu env | grep VAR
VAR1=value1
VAR2=value2
```

When running the command, the Docker CLI client checks the value the variable
has in your local environment and passes it to the container.
If no `=` is provided and that variable is not exported in your local
environment, the variable isn't set in the container.

You can also load the environment variables from a file. This file should use
the syntax `<variable>=value` (which sets the variable to the given value) or
`<variable>` (which takes the value from the local environment), and `#` for comments.
Additionally, it's important to note that lines beginning with `#` are treated as line comments
and are ignored, whereas a `#` appearing anywhere else in a line is treated as part of the variable value.

```console
$ cat env.list
# This is a comment
VAR1=value1
VAR2=value2
USER

$ docker run --env-file env.list ubuntu env | grep -E 'VAR|USER'
VAR1=value1
VAR2=value2
USER=jonzeolla
```

### <a name="label"></a> Set metadata on container (-l, --label, --label-file)

A label is a `key=value` pair that applies metadata to a container. To label a container with two labels:

```console
$ docker run -l my-label --label com.example.foo=bar ubuntu bash
```

The `my-label` key doesn't specify a value so the label defaults to an empty
string (`""`). To add multiple labels, repeat the label flag (`-l` or `--label`).

The `key=value` must be unique to avoid overwriting the label value. If you
specify labels with identical keys but different values, each subsequent value
overwrites the previous. Docker uses the last `key=value` you supply.

Use the `--label-file` flag to load multiple labels from a file. Delimit each
label in the file with an EOL mark. The example below loads labels from a
labels file in the current directory:

```console
$ docker run --label-file ./labels ubuntu bash
```

The label-file format is similar to the format for loading environment
variables. (Unlike environment variables, labels are not visible to processes
running inside a container.) The following example shows a label-file
format:

```console
com.example.label1="a label"

# this is a comment
com.example.label2=another\ label
com.example.label3
```

You can load multiple label-files by supplying multiple  `--label-file` flags.

For additional information on working with labels, see [*Labels - custom
metadata in Docker*](https://docs.docker.com/config/labels-custom-metadata/) in
the Docker User Guide.

### <a name="network"></a> Connect a container to a network (--network)

To start a container and connect it to a network, use the `--network` option.

The following commands create a network named `my-net` and adds a `busybox` container
to the `my-net` network.

```console
$ docker network create my-net
$ docker run -itd --network=my-net busybox
```

You can also choose the IP addresses for the container with `--ip` and `--ip6`
flags when you start the container on a user-defined network. To assign a
static IP to containers, you must specify subnet block for the network.

```console
$ docker network create --subnet 192.0.2.0/24 my-net
$ docker run -itd --network=my-net --ip=192.0.2.69 busybox
```

If you want to add a running container to a network use the `docker network connect` subcommand.

You can connect multiple containers to the same network. Once connected, the
containers can communicate using only another container's IP address
or name. For `overlay` networks or custom plugins that support multi-host
connectivity, containers connected to the same multi-host network but launched
from different Engines can also communicate in this way.

> **Note**
>
> The default bridge network only allow containers to communicate with each other using
> internal IP addresses. User-created bridge networks provide DNS resolution between
> containers using container names.

You can disconnect a container from a network using the `docker network
disconnect` command.

For more information on connecting a container to a network when using the `run` command, see the ["*Docker network overview*"](https://docs.docker.com/network/).

### <a name="volumes-from"></a> Mount volumes from container (--volumes-from)

```console
$ docker run --volumes-from 777f7dc92da7 --volumes-from ba8c0c54f0f2:ro -i -t ubuntu pwd
```

The `--volumes-from` flag mounts all the defined volumes from the referenced
containers. You can specify more than one container by repetitions of the `--volumes-from`
argument. The container ID may be optionally suffixed with `:ro` or `:rw` to
mount the volumes in read-only or read-write mode, respectively. By default,
Docker mounts the volumes in the same mode (read write or read only) as
the reference container.

Labeling systems like SELinux require placing proper labels on volume
content mounted into a container. Without a label, the security system might
prevent the processes running inside the container from using the content. By
default, Docker does not change the labels set by the OS.

To change the label in the container context, you can add either of two suffixes
`:z` or `:Z` to the volume mount. These suffixes tell Docker to relabel file
objects on the shared volumes. The `z` option tells Docker that two containers
share the volume content. As a result, Docker labels the content with a shared
content label. Shared volume labels allow all containers to read/write content.
The `Z` option tells Docker to label the content with a private unshared label.
Only the current container can use a private volume.

### <a name="attach"></a> Attach to STDIN/STDOUT/STDERR (-a, --attach)

The `--attach` (or `-a`) flag tells `docker run` to bind to the container's
`STDIN`, `STDOUT` or `STDERR`. This makes it possible to manipulate the output
and input as needed.

```console
$ echo "test" | docker run -i -a stdin ubuntu cat -
```

This pipes data into a container and prints the container's ID by attaching
only to the container's `STDIN`.

```console
$ docker run -a stderr ubuntu echo test
```

This isn't going to print anything to the console unless there's an error because output
is only attached to the `STDERR` of the container. The container's logs
still store what's written to `STDERR` and `STDOUT`.

```console
$ cat somefile | docker run -i -a stdin mybuilder dobuild
```

This example shows a way of using `--attach` to pipe a file into a container.
The command prints the container's ID after the build completes and you can retrieve
the build logs using `docker logs`. This is
useful if you need to pipe a file or something else into a container and
retrieve the container's ID once the container has finished running.

See also [the `docker cp` command](cp.md).

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
containers, see [**Configuration file** section](cli.md#configuration-files).

### <a name="device"></a> Add host device to container (--device)

```console
$ docker run -it --rm \
    --device=/dev/sdc:/dev/xvdc \
    --device=/dev/sdd \
    --device=/dev/zero:/dev/foobar \
    ubuntu ls -l /dev/{xvdc,sdd,foobar}

brw-rw---- 1 root disk 8, 2 Feb  9 16:05 /dev/xvdc
brw-rw---- 1 root disk 8, 3 Feb  9 16:05 /dev/sdd
crw-rw-rw- 1 root root 1, 5 Feb  9 16:05 /dev/foobar
```

It's often necessary to directly expose devices to a container. The `--device`
option enables that. For example, adding a specific block storage device or loop
device or audio device to an otherwise unprivileged container
(without the `--privileged` flag) and have the application directly access it.

By default, the container is able to `read`, `write` and `mknod` these devices.
This can be overridden using a third `:rwm` set of options to each `--device`
flag. If the container is running in privileged mode, then Docker ignores the
specified permissions.

```console
$ docker run --device=/dev/sda:/dev/xvdc --rm -it ubuntu fdisk  /dev/xvdc

Command (m for help): q
$ docker run --device=/dev/sda:/dev/xvdc:r --rm -it ubuntu fdisk  /dev/xvdc
You will not be able to write the partition table.

Command (m for help): q

$ docker run --device=/dev/sda:/dev/xvdc:rw --rm -it ubuntu fdisk  /dev/xvdc

Command (m for help): q

$ docker run --device=/dev/sda:/dev/xvdc:m --rm -it ubuntu fdisk  /dev/xvdc
fdisk: unable to open /dev/xvdc: Operation not permitted
```

> **Note**
>
> The `--device` option cannot be safely used with ephemeral devices. You shouldn't 
> add block devices that may be removed to untrusted containers with `--device`.

For Windows, the format of the string passed to the `--device` option is in
the form of `--device=<IdType>/<Id>`. Beginning with Windows Server 2019
and Windows 10 October 2018 Update, Windows only supports an IdType of
`class` and the Id as a [device interface class
GUID](https://docs.microsoft.com/en-us/windows-hardware/drivers/install/overview-of-device-interface-classes).
Refer to the table defined in the [Windows container
docs](https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/hardware-devices-in-containers)
for a list of container-supported device interface class GUIDs.

If you specify this option for a process-isolated Windows container, Docker makes
_all_ devices that implement the requested device interface class GUID
available in the container. For example, the command below makes all COM
ports on the host visible in the container.

```powershell
PS C:\> docker run --device=class/86E0D1E0-8089-11D0-9CE4-08003E301F73 mcr.microsoft.com/windows/servercore:ltsc2019
```

> **Note**
>
> The `--device` option is only supported on process-isolated Windows containers,
> and produces an error if the container isolation is `hyperv`.

### <a name="device-cgroup-rule"></a> Using dynamically created devices (--device-cgroup-rule)

Docker assigns devices available to a container at creation time. The
assigned devices are added to the cgroup.allow file and
created into the container when it runs. This poses a problem when
you need to add a new device to running container.

One solution is to add a more permissive rule to a container
allowing it access to a wider range of devices. For example, supposing
the container needs access to a character device with major `42` and
any number of minor numbers (added as new devices appear), add the
following rule:

```console
$ docker run -d --device-cgroup-rule='c 42:* rmw' --name my-container my-image
```

Then, a user could ask `udev` to execute a script that would `docker exec my-container mknod newDevX c 42 <minor>`
the required device when it is added.

> **Note**: You still need to explicitly add initially present devices to the
> `docker run` / `docker create` command.

### <a name="gpus"></a> Access an NVIDIA GPU

The `--gpus` flag allows you to access NVIDIA GPU resources. First you need to
install the [nvidia-container-runtime](https://nvidia.github.io/nvidia-container-runtime/).

Read [Specify a container's resources](https://docs.docker.com/config/containers/resource_constraints/)
for more information.

To use `--gpus`, specify which GPUs (or all) to use. If you provide no value, Docker uses all
available GPUs. The example below exposes all available GPUs.

```console
$ docker run -it --rm --gpus all ubuntu nvidia-smi
```

Use the `device` option to specify GPUs. The example below exposes a specific
GPU.

```console
$ docker run -it --rm --gpus device=GPU-3a23c669-1f69-c64e-cf85-44e9b07e7a2a ubuntu nvidia-smi
```

The example below exposes the first and third GPUs.

```console
$ docker run -it --rm --gpus '"device=0,2"' nvidia-smi
```

### <a name="restart"></a> Restart policies (--restart)

Use the `--restart` flag to specify a container's *restart policy*. A restart
policy controls whether the Docker daemon restarts a container after exit.
Docker supports the following restart policies:

| Policy                     | Result                                                                                                                                                                                                                                                           |
|:---------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `no`                       | Do not automatically restart the container when it exits. This is the default.                                                                                                                                                                                   |
| `on-failure[:max-retries]` | Restart only if the container exits with a non-zero exit status. Optionally, limit the number of restart retries the Docker daemon attempts.                                                                                                                     |
| `unless-stopped`           | Restart the container unless it's explicitly stopped or Docker itself is stopped or restarted.                                                                                                                                                                  |
| `always`                   | Always restart the container regardless of the exit status. When you specify always, the Docker daemon tries to restart the container indefinitely. The container always starts on daemon startup, regardless of the current state of the container. |

```console
$ docker run --restart=always redis
```

This will run the `redis` container with a restart policy of **always**
so that if the container exits, Docker restarts it.

You can find more detailed information on restart policies in the
[Restart Policies (--restart)](../run.md#restart-policies---restart)
section of the Docker run reference page.

### <a name="add-host"></a> Add entries to container hosts file (--add-host)

You can add other hosts into a container's `/etc/hosts` file by using one or
more `--add-host` flags. This example adds a static address for a host named
`docker`:

```console
$ docker run --add-host=docker:93.184.216.34 --rm -it alpine

/ # ping docker
PING docker (93.184.216.34): 56 data bytes
64 bytes from 93.184.216.34: seq=0 ttl=37 time=93.052 ms
64 bytes from 93.184.216.34: seq=1 ttl=37 time=92.467 ms
64 bytes from 93.184.216.34: seq=2 ttl=37 time=92.252 ms
^C
--- docker ping statistics ---
4 packets transmitted, 4 packets received, 0% packet loss
round-trip min/avg/max = 92.209/92.495/93.052 ms
```

The `--add-host` flag supports a special `host-gateway` value that resolves to
the internal IP address of the host. This is useful when you want containers to
connect to services running on the host machine.

It's conventional to use `host.docker.internal` as the hostname referring to
`host-gateway`. Docker Desktop automatically resolves this hostname, see
[Explore networking features](https://docs.docker.com/desktop/networking/#i-want-to-connect-from-a-container-to-a-service-on-the-host).

The following example shows how the special `host-gateway` value works. The
example runs an HTTP server that serves a file from host to container over the
`host.docker.internal` hostname, which resolves to the host's internal IP.

```console
$ echo "hello from host!" > ./hello
$ python3 -m http.server 8000
Serving HTTP on 0.0.0.0 port 8000 (http://0.0.0.0:8000/) ...
$ docker run \
  --add-host host.docker.internal:host-gateway \
  curlimages/curl -s host.docker.internal:8000/hello
hello from host!
```

### <a name="ulimit"></a> Set ulimits in container (--ulimit)

Since setting `ulimit` settings in a container requires extra privileges not
available in the default container, you can set these using the `--ulimit` flag.
Specify `--ulimit` with a soft and hard limit in the format
`<type>=<soft limit>[:<hard limit>]`. For example:

```console
$ docker run --ulimit nofile=1024:1024 --rm debian sh -c "ulimit -n"
1024
```

> **Note**
>
> If you don't provide a hard limit value, Docker uses the soft limit value
> for both values. If you don't provide any values, they are inherited from
> the default `ulimits` set on the daemon.

> **Note**
>
> The `as` option is deprecated.
> In other words, the following script is not supported:
>
> ```console
> $ docker run -it --ulimit as=1024 fedora /bin/bash
> ```

Docker sends the values to the appropriate OS `syscall` and doesn't perform any byte conversion.
Take this into account when setting the values.

#### For `nproc` usage

Be careful setting `nproc` with the `ulimit` flag as Linux uses `nproc` to set the
maximum number of processes available to a user, not to a container. For example, start four
containers with `daemon` user:

```console
$ docker run -d -u daemon --ulimit nproc=3 busybox top

$ docker run -d -u daemon --ulimit nproc=3 busybox top

$ docker run -d -u daemon --ulimit nproc=3 busybox top

$ docker run -d -u daemon --ulimit nproc=3 busybox top
```

The 4th container fails and reports a "[8] System error: resource temporarily unavailable" error.
This fails because the caller set `nproc=3` resulting in the first three containers using up
the three processes quota set for the `daemon` user.

### <a name="stop-signal"></a> Stop container with signal (--stop-signal)

The `--stop-signal` flag sends the system call signal to the
container to exit. This signal can be a signal name in the format `SIG<NAME>`,
for instance `SIGKILL`, or an unsigned number that matches a position in the
kernel's syscall table, for instance `9`.

The default value is defined by [`STOPSIGNAL`](https://docs.docker.com/engine/reference/builder/#stopsignal)
in the image, or `SIGTERM` if the image has no `STOPSIGNAL` defined.

### <a name="security-opt"></a> Optional security options (--security-opt)

On Windows, you can use this flag to specify the `credentialspec` option.
The `credentialspec` must be in the format `file://spec.txt` or `registry://keyname`.

### <a name="stop-timeout"></a> Stop container with timeout (--stop-timeout)

The `--stop-timeout` flag sets the number of seconds to wait for the container
to stop after sending the pre-defined (see `--stop-signal`) system call signal.
If the container does not exit after the timeout elapses, it's forcibly killed
with a `SIGKILL` signal.

If you set `--stop-timeout` to `-1`, no timeout is applied, and the daemon
waits indefinitely for the container to exit.

The Daemon determines the default, and is 10 seconds for Linux containers,
and 30 seconds for Windows containers.

### <a name="isolation"></a> Specify isolation technology for container (--isolation)

This option is useful in situations where you are running Docker containers on
Windows. The `--isolation=<value>` option sets a container's isolation technology.
On Linux, the only supported is the `default` option which uses Linux namespaces.
These two commands are equivalent on Linux:

```console
$ docker run -d busybox top
$ docker run -d --isolation default busybox top
```

On Windows, `--isolation` can take one of these values:

| Value     | Description                                                                                |
|:----------|:-------------------------------------------------------------------------------------------|
| `default` | Use the value specified by the Docker daemon's `--exec-opt` or system default (see below). |
| `process` | Shared-kernel namespace isolation.                                                         |
| `hyperv`  | Hyper-V hypervisor partition-based isolation.                                              |

The default isolation on Windows server operating systems is `process`, and `hyperv`
on Windows client operating systems, such as Windows 10. Process isolation has better
performance, but requires that the image and host use the same kernel version.

On Windows server, assuming the default configuration, these commands are equivalent
and result in `process` isolation:

```powershell
PS C:\> docker run -d microsoft/nanoserver powershell echo process
PS C:\> docker run -d --isolation default microsoft/nanoserver powershell echo process
PS C:\> docker run -d --isolation process microsoft/nanoserver powershell echo process
```

If you have set the `--exec-opt isolation=hyperv` option on the Docker `daemon`, or
are running against a Windows client-based daemon, these commands are equivalent and
result in `hyperv` isolation:

```powershell
PS C:\> docker run -d microsoft/nanoserver powershell echo hyperv
PS C:\> docker run -d --isolation default microsoft/nanoserver powershell echo hyperv
PS C:\> docker run -d --isolation hyperv microsoft/nanoserver powershell echo hyperv
```

### <a name="memory"></a> Specify hard limits on memory available to containers (-m, --memory)

These parameters always set an upper limit on the memory available to the container. Linux sets this
on the cgroup and applications in a container can query it at `/sys/fs/cgroup/memory/memory.limit_in_bytes`.

On Windows, this affects containers differently depending on what type of isolation you use.

- With `process` isolation, Windows reports the full memory of the host system, not the limit to applications running inside the container

    ```powershell
    PS C:\> docker run -it -m 2GB --isolation=process microsoft/nanoserver powershell Get-ComputerInfo *memory*

    CsTotalPhysicalMemory      : 17064509440
    CsPhyicallyInstalledMemory : 16777216
    OsTotalVisibleMemorySize   : 16664560
    OsFreePhysicalMemory       : 14646720
    OsTotalVirtualMemorySize   : 19154928
    OsFreeVirtualMemory        : 17197440
    OsInUseVirtualMemory       : 1957488
    OsMaxProcessMemorySize     : 137438953344
    ```

- With `hyperv` isolation, Windows creates a utility VM that is big enough to hold the memory limit, plus the minimal OS needed to host the container. That size is reported as "Total Physical Memory."

    ```powershell
    PS C:\> docker run -it -m 2GB --isolation=hyperv microsoft/nanoserver powershell Get-ComputerInfo *memory*

    CsTotalPhysicalMemory      : 2683355136
    CsPhyicallyInstalledMemory :
    OsTotalVisibleMemorySize   : 2620464
    OsFreePhysicalMemory       : 2306552
    OsTotalVirtualMemorySize   : 2620464
    OsFreeVirtualMemory        : 2356692
    OsInUseVirtualMemory       : 263772
    OsMaxProcessMemorySize     : 137438953344
    ```

### <a name="sysctl"></a> Configure namespaced kernel parameters (sysctls) at runtime (--sysctl)

The `--sysctl` sets namespaced kernel parameters (sysctls) in the
container. For example, to turn on IP forwarding in the containers
network namespace, run this command:

```console
$ docker run --sysctl net.ipv4.ip_forward=1 someimage
```

> **Note**
>
> Not all sysctls are namespaced. Docker does not support changing sysctls
> inside of a container that also modify the host system. As the kernel
> evolves we expect to see more sysctls become namespaced.


#### Currently supported sysctls

IPC Namespace:

- `kernel.msgmax`, `kernel.msgmnb`, `kernel.msgmni`, `kernel.sem`,
  `kernel.shmall`, `kernel.shmmax`, `kernel.shmmni`, `kernel.shm_rmid_forced`.
- Sysctls beginning with `fs.mqueue.*`
- If you use the `--ipc=host` option these sysctls are not allowed.

Network Namespace:

- Sysctls beginning with `net.*`
- If you use the `--network=host` option using these sysctls are not allowed.

## Command internals

The `docker run` command is equivalent to the following API calls:

- `/<API version>/containers/create`
  - If that call returns a 404 (image not found), and depending on the `--pull` option ("always", "missing", "never") the call can trigger a `docker pull <image>`.
- `/containers/create` again after pulling the image.
- `/containers/(id)/start` to start the container.
- `/containers/(id)/attach` to attach to the container when starting with the `-it` flags for interactive containers.
