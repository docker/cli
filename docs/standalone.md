# Standalone mode (no daemon)

Setting `DOCKER_STANDALONE=1` makes the `docker` CLI run containers itself,
without a Docker Engine. Neither `dockerd` nor a `containerd` daemon is
needed: the CLI embeds the containerd libraries (content store, metadata
store, snapshotters, image puller and OCI spec generation) and drives an OCI
runtime (`runc` or `crun`) directly.

```console
$ export DOCKER_STANDALONE=1
$ docker run --rm alpine echo hello
hello
```

The same commands, flags and output formats are used as against a daemon, so
existing scripts keep working for the supported subset described below.

## How it works

A daemon normally keeps container state in memory and supervises container
processes. Without one, both jobs are handled per container:

- **State** lives in a containerd metadata database (bbolt) under the storage
  root, together with the content store, snapshots and volumes. Each CLI
  invocation opens the database for the duration of the operation, so several
  commands can run one after another.
- **Supervision** is done by `containerd-shim-runc-v2`, the same per-container
  shim that containerd uses. The CLI starts it through containerd's runtime
  library; it daemonizes itself, owns the container's I/O and reports the exit
  status. Because the shim outlives the CLI, `docker run -d` works and later
  commands (`docker stop`, `docker exec`, ...) reconnect to it.
- **Logs, attach and exit status** are handled by a small helper process that
  the CLI starts for each container (the `docker` binary re-executed in a
  logging mode). It writes the container's output to a `json-file` log,
  serves `docker attach` clients, forwards their input to the container's
  stdin, and records the exit status in the metadata store when the container
  exits.

Only two helper processes per container are involved (the shim and the logging
helper), plus one network helper when a network namespace is used. Nothing
runs when no container is running.

## Requirements

- Linux.
- An OCI runtime: `runc` (preferred) or `crun`.
  Older `crun` releases reject the runtime-spec version emitted by containerd;
  use `DOCKER_STANDALONE_RUNTIME=runc` if container creation fails with
  `unknown version specified`.
- `containerd-shim-runc-v2` in `PATH` (shipped with containerd and nerdctl
  packages).
- For rootless mode: `newuidmap` and `newgidmap` (the `uidmap` package) with
  subordinate ID ranges in `/etc/subuid` and `/etc/subgid`, and `pasta`
  (the `passt` package) for container networking.

## Rootful and rootless

Both are supported and selected automatically from the effective user:

|                | rootful                    | rootless                                |
| -------------- | -------------------------- | --------------------------------------- |
| storage root   | `/var/lib/docker-standalone` | `$XDG_DATA_HOME/docker-standalone`    |
| runtime state  | `/run/docker-standalone`   | `$XDG_RUNTIME_DIR/docker-standalone`    |
| snapshotter    | `overlayfs`, else `native` | `overlayfs` (kernel ≥ 5.11), else `native` |
| networking     | CNI bridge if the plugins are installed, else `pasta` | `pasta`, else `slirp4netns` |
| cgroups        | cgroupfs                   | only with a delegated cgroup (systemd user session) |

In rootless mode the CLI re-executes itself in a new user namespace, mapping
the caller's subordinate ID ranges with `newuidmap`/`newgidmap`, and in a new
mount namespace so that image layers can be mounted. Container processes then
run as root inside that namespace and unprivileged on the host.

Resource limits (`--memory`, `--cpus`, `--pids-limit`) and `docker pause`
require a cgroup. Rootless containers only get one when the user's systemd
instance can create transient units (`systemd --user` running, cgroup v2); the
container is placed in `user.slice`. Without it, containers still run but
limits are not enforced.

## Configuration

| Environment variable             | Description                                                     |
| -------------------------------- | --------------------------------------------------------------- |
| `DOCKER_STANDALONE`              | Set to `1` to enable standalone mode.                           |
| `DOCKER_STANDALONE_ROOT`         | Storage root (images, containers, volumes).                     |
| `DOCKER_STANDALONE_RUN`          | Runtime state directory (sockets, bundles, FIFOs, namespaces).  |
| `DOCKER_STANDALONE_RUNTIME`      | OCI runtime binary or path (`runc`, `crun`, ...).               |
| `DOCKER_STANDALONE_SNAPSHOTTER`  | `overlayfs` or `native`. Recorded on first use for that root.   |
| `DOCKER_STANDALONE_NETWORK`      | Backend for the default network: `bridge`, `pasta`, `slirp4netns`, `host` or `none`. |

Shim diagnostics are written to `<runtime state>/shim.log` instead of the
terminal; run with `--debug` to see them on stderr.

## What is supported

Images
: `pull`, `push`, `images` (including `--tree`), `inspect`, `history`, `tag`,
  `rmi`, `prune`.

Containers
: `create`, `run` (foreground, `-d`, `-i`, `-t`), `start`, `stop`, `restart`,
  `kill`, `pause`, `unpause`, `rm`, `prune`, `ps`, `inspect`, `logs`
  (including `-f`, `--tail`, `--since`, `--timestamps`), `exec`, `attach`,
  `top`, `rename`, `wait`, `port`.

Configuration
: environment, working directory, user, hostname, entrypoint and command,
  labels, stop signal and timeout, `--read-only`, masked and readonly paths,
  capabilities, `--privileged`, seccomp and AppArmor profiles, devices,
  sysctls, ulimits, resource limits, `--network bridge|host|none`,
  published ports, `--dns`, `--add-host`, bind mounts, named and anonymous
  volumes, `tmpfs`.

Volumes and networks
: `volume create|ls|inspect|rm|prune` (`local` driver), `network ls|inspect`
  for the built-in `bridge`, `host` and `none` networks.

System
: `version`, `info`.

## Limitations

- **No build.** `docker build` needs BuildKit; use `buildctl` or `docker
  buildx` against a builder, and load the resulting image.
- **No Swarm, plugins, checkpoints, or `docker events`.** These commands
  report that they are not supported.
- **No user-defined networks**; the built-in `bridge`, `host` and `none`
  networks are available. `--network container:<id>` is not implemented.
- **Restart policies are not enforced**: nothing runs to restart a container
  after it exits.
- **No health checks**, for the same reason.
- `docker cp`, `docker commit`, `docker export`, `docker save` and
  `docker load` are not implemented yet.
- Containers are not restarted after a reboot, and their state is reconciled
  on the next command.
- Published ports are not supported with the `slirp4netns` backend; use
  `pasta`.
- In rootless mode, `docker top` may only report the container's main process
  because each CLI invocation runs in its own user namespace.

## Troubleshooting

`docker: creating container task: ... unknown version specified`
: The OCI runtime is too old for the runtime-spec version used. Use
  `DOCKER_STANDALONE_RUNTIME=runc` or upgrade `crun`.

`failed to start shim: ... containerd-shim-runc-v2`
: The shim binary is not in `PATH`.

`WARNING: no subordinate UID/GID ranges found for the current user`
: Add ranges for the user to `/etc/subuid` and `/etc/subgid` and install
  `newuidmap`/`newgidmap`; without them a single ID is mapped, which breaks
  images that use other users.

Resource limits have no effect (rootless)
: The current cgroup is not delegated. Run inside a systemd user session, for
  example `systemd-run --user --scope docker run ...`.
