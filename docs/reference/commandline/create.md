---
title: "create"
description: "The create command description and usage"
keywords: "docker, create, container"
---

# create

Creates a new container.

```markdown
Usage:  docker create [OPTIONS] IMAGE [COMMAND] [ARG...]

Create a new container

Options:
      --add-host value                Add a custom host-to-IP mapping (host:ip) (default [])
  -a, --attach value                  Attach to STDIN, STDOUT or STDERR (default [])
      --blkio-weight value            Block IO (relative weight), between 10 and 1000
      --blkio-weight-device value     Block IO weight (relative device weight) (default [])
      --cap-add value                 Add Linux capabilities (default [])
      --cap-drop value                Drop Linux capabilities (default [])
      --cgroupns string               Cgroup namespace to use
                                      'host':    Run the container in the Docker host's cgroup namespace
                                      'private': Run the container in its own private cgroup namespace
                                      '':        Use the default Docker daemon cgroup namespace specified by the `--default-cgroupns-mode` option
      --cgroup-parent string          Optional parent cgroup for the container
      --cidfile string                Write the container ID to the file
      --cpu-count int                 The number of CPUs available for execution by the container.
                                      Windows daemon only. On Windows Server containers, this is
                                      approximated as a percentage of total CPU usage.
      --cpu-percent int               CPU percent (Windows only)
      --cpu-period int                Limit CPU CFS (Completely Fair Scheduler) period
      --cpu-quota int                 Limit CPU CFS (Completely Fair Scheduler) quota
  -c, --cpu-shares int                CPU shares (relative weight)
      --cpus NanoCPUs                 Number of CPUs (default 0.000)
      --cpu-rt-period int             Limit the CPU real-time period in microseconds
      --cpu-rt-runtime int            Limit the CPU real-time runtime in microseconds
      --cpuset-cpus string            CPUs in which to allow execution (0-3, 0,1)
      --cpuset-mems string            MEMs in which to allow execution (0-3, 0,1)
      --device value                  Add a host device to the container (default [])
      --device-cgroup-rule value      Add a rule to the cgroup allowed devices list
      --device-read-bps value         Limit read rate (bytes per second) from a device (default [])
      --device-read-iops value        Limit read rate (IO per second) from a device (default [])
      --device-write-bps value        Limit write rate (bytes per second) to a device (default [])
      --device-write-iops value       Limit write rate (IO per second) to a device (default [])
      --disable-content-trust         Skip image verification (default true)
      --dns value                     Set custom DNS servers (default [])
      --dns-option value              Set DNS options (default [])
      --dns-search value              Set custom DNS search domains (default [])
      --domainname string             Container NIS domain name
      --entrypoint string             Overwrite the default ENTRYPOINT of the image
  -e, --env value                     Set environment variables (default [])
      --env-file value                Read in a file of environment variables (default [])
      --expose value                  Expose a port or a range of ports (default [])
      --group-add value               Add additional groups to join (default [])
      --health-cmd string             Command to run to check health
      --health-interval duration      Time between running the check (ns|us|ms|s|m|h) (default 0s)
      --health-retries int            Consecutive failures needed to report unhealthy
      --health-timeout duration       Maximum time to allow one check to run (ns|us|ms|s|m|h) (default 0s)
      --health-start-period duration  Start period for the container to initialize before counting retries towards unstable (ns|us|ms|s|m|h) (default 0s)
      --help                          Print usage
  -h, --hostname string               Container host name
      --init                          Run an init inside the container that forwards signals and reaps processes
  -i, --interactive                   Keep STDIN open even if not attached
      --io-maxbandwidth string        Maximum IO bandwidth limit for the system drive (Windows only)
      --io-maxiops uint               Maximum IOps limit for the system drive (Windows only)
      --ip string                     IPv4 address (e.g., 172.30.100.104)
      --ip6 string                    IPv6 address (e.g., 2001:db8::33)
      --ipc string                    IPC namespace to use
      --isolation string              Container isolation technology
      --kernel-memory string          Kernel memory limit
  -l, --label value                   Set meta data on a container (default [])
      --label-file value              Read in a line delimited file of labels (default [])
      --link value                    Add link to another container (default [])
      --link-local-ip value           Container IPv4/IPv6 link-local addresses (default [])
      --log-driver string             Logging driver for the container
      --log-opt value                 Log driver options (default [])
      --mac-address string            Container MAC address (e.g., 92:d0:c6:0a:29:33)
  -m, --memory string                 Memory limit
      --memory-reservation string     Memory soft limit
      --memory-swap string            Swap limit equal to memory plus swap: '-1' to enable unlimited swap
      --memory-swappiness int         Tune container memory swappiness (0 to 100) (default -1)
      --mount value                   Attach a filesystem mount to the container (default [])
      --name string                   Assign a name to the container
      --network-alias value           Add network-scoped alias for the container (default [])
      --network string                Connect a container to a network (default "default")
                                      'bridge': create a network stack on the default Docker bridge
                                      'none': no networking
                                      'container:<name|id>': reuse another container's network stack
                                      'host': use the Docker host network stack
                                      '<network-name>|<network-id>': connect to a user-defined network
      --no-healthcheck                Disable any container-specified HEALTHCHECK
      --oom-kill-disable              Disable OOM Killer
      --oom-score-adj int             Tune host's OOM preferences (-1000 to 1000)
      --pid string                    PID namespace to use
      --pids-limit int                Tune container pids limit (set -1 for unlimited), kernel >= 4.3
      --privileged                    Give extended privileges to this container
  -p, --publish value                 Publish a container's port(s) to the host (default [])
  -P, --publish-all                   Publish all exposed ports to random ports
      --pull string                   Pull image before creating ("always"|"missing"|"never") (default "missing")
      --read-only                     Mount the container's root filesystem as read only
      --restart string                Restart policy to apply when a container exits (default "no")
                                      Possible values are: no, on-failure[:max-retry], always, unless-stopped
      --rm                            Automatically remove the container when it exits
      --runtime string                Runtime to use for this container
      --security-opt value            Security Options (default [])
      --shm-size bytes                Size of /dev/shm
                                      The format is `<number><unit>`. `number` must be greater than `0`.
                                      Unit is optional and can be `b` (bytes), `k` (kilobytes), `m` (megabytes),
                                      or `g` (gigabytes). If you omit the unit, the system uses bytes.
      --stop-signal string            Signal to stop a container (default "SIGTERM")
      --stop-timeout int              Timeout (in seconds) to stop a container
      --storage-opt value             Storage driver options for the container (default [])
      --sysctl value                  Sysctl options (default map[])
      --tmpfs value                   Mount a tmpfs directory (default [])
  -t, --tty                           Allocate a pseudo-TTY
      --ulimit value                  Ulimit options (default [])
  -u, --user string                   Username or UID (format: <name|uid>[:<group|gid>])
      --userns string                 User namespace to use
                                      'host': Use the Docker host user namespace
                                      '': Use the Docker daemon user namespace specified by `--userns-remap` option.
      --uts string                    UTS namespace to use
  -v, --volume value                  Bind mount a volume (default []). The format
                                      is `[host-src:]container-dest[:<options>]`.
                                      The comma-delimited `options` are [rw|ro],
                                      [z|Z], [[r]shared|[r]slave|[r]private],
                                      [delegated|cached|consistent], and
                                      [nocopy]. The 'host-src' is an absolute path
                                      or a name value.
      --volume-driver string          Optional volume driver for the container
      --volumes-from value            Mount volumes from the specified container(s) (default [])
  -w, --workdir string                Working directory inside the container
```

## Description

The `docker container create` (or shorthand: `docker create`) command creates a
new container from the specified image, without starting it.

When creating a container, the docker daemon creates a writeable container layer
over the specified image and prepares it for running the specified command.  The
container ID is then printed to `STDOUT`.  This is similar to `docker run -d`
except the container is never started. You can then use the `docker container start`
(or shorthand: `docker start`) command to start the container at any point.

This is useful when you want to set up a container configuration ahead of time
so that it is ready to start when you need it. The initial status of the
new container is `created`.

The `docker create` command shares most of its options with the `docker run`
command (which performs a `docker create` before starting it). Refer to the
[`docker run` command](run.md) section and the [Docker run reference](../run.md)
for details on the available flags and options.

## Examples

### Create and start a container

The following example creates an interactive container with a pseudo-TTY attached,
then starts the container and attaches to it:

```console
$ docker container create -i -t --name mycontainer alpine
6d8af538ec541dd581ebc2a24153a28329acb5268abe5ef868c1f1a261221752

$ docker container start --attach -i mycontainer
/ # echo hello world
hello world
```

The above is the equivalent of a `docker run`:

```console
$ docker run -it --name mycontainer2 alpine
/ # echo hello world
hello world
```

### Initialize volumes

Container volumes are initialized during the `docker create` phase
(i.e., `docker run` too). For example, this allows you to `create` the `data`
volume container, and then use it from another container:

```console
$ docker create -v /data --name data ubuntu

240633dfbb98128fa77473d3d9018f6123b99c454b3251427ae190a7d951ad57

$ docker run --rm --volumes-from data ubuntu ls -la /data

total 8
drwxr-xr-x  2 root root 4096 Dec  5 04:10 .
drwxr-xr-x 48 root root 4096 Dec  5 04:11 ..
```

Similarly, `create` a host directory bind mounted volume container, which can
then be used from the subsequent container:

```console
$ docker create -v /home/docker:/docker --name docker ubuntu

9aa88c08f319cd1e4515c3c46b0de7cc9aa75e878357b1e96f91e2c773029f03

$ docker run --rm --volumes-from docker ubuntu ls -la /docker

total 20
drwxr-sr-x  5 1000 staff  180 Dec  5 04:00 .
drwxr-xr-x 48 root root  4096 Dec  5 04:13 ..
-rw-rw-r--  1 1000 staff 3833 Dec  5 04:01 .ash_history
-rw-r--r--  1 1000 staff  446 Nov 28 11:51 .ashrc
-rw-r--r--  1 1000 staff   25 Dec  5 04:00 .gitconfig
drwxr-sr-x  3 1000 staff   60 Dec  1 03:28 .local
-rw-r--r--  1 1000 staff  920 Nov 28 11:51 .profile
drwx--S---  2 1000 staff  460 Dec  5 00:51 .ssh
drwxr-xr-x 32 1000 staff 1140 Dec  5 04:01 docker
```
