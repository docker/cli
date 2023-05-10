---
title: "Use the Docker command line"
description: "Docker's CLI command description and usage"
keywords: "Docker, Docker documentation, CLI, command line, config.json, CLI configuration file"
redirect_from:
  - /reference/commandline/cli/
  - /engine/reference/commandline/engine/
  - /engine/reference/commandline/engine_activate/
  - /engine/reference/commandline/engine_check/
  - /engine/reference/commandline/engine_update/
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# docker

To list available commands, either run `docker` with no parameters
or execute `docker help`:

<!---MARKER_GEN_START-->
The base command for the Docker CLI.

### Subcommands

| Name                          | Description                                                                   |
|:------------------------------|:------------------------------------------------------------------------------|
| [`attach`](attach.md)         | Attach local standard input, output, and error streams to a running container |
| [`build`](build.md)           | Build an image from a Dockerfile                                              |
| [`builder`](builder.md)       | Manage builds                                                                 |
| [`checkpoint`](checkpoint.md) | Manage checkpoints                                                            |
| [`commit`](commit.md)         | Create a new image from a container's changes                                 |
| [`config`](config.md)         | Manage Swarm configs                                                          |
| [`container`](container.md)   | Manage containers                                                             |
| [`context`](context.md)       | Manage contexts                                                               |
| [`cp`](cp.md)                 | Copy files/folders between a container and the local filesystem               |
| [`create`](create.md)         | Create a new container                                                        |
| [`diff`](diff.md)             | Inspect changes to files or directories on a container's filesystem           |
| [`events`](events.md)         | Get real time events from the server                                          |
| [`exec`](exec.md)             | Execute a command in a running container                                      |
| [`export`](export.md)         | Export a container's filesystem as a tar archive                              |
| [`history`](history.md)       | Show the history of an image                                                  |
| [`image`](image.md)           | Manage images                                                                 |
| [`images`](images.md)         | List images                                                                   |
| [`import`](import.md)         | Import the contents from a tarball to create a filesystem image               |
| [`info`](info.md)             | Display system-wide information                                               |
| [`inspect`](inspect.md)       | Return low-level information on Docker objects                                |
| [`kill`](kill.md)             | Kill one or more running containers                                           |
| [`load`](load.md)             | Load an image from a tar archive or STDIN                                     |
| [`login`](login.md)           | Log in to a registry                                                          |
| [`logout`](logout.md)         | Log out from a registry                                                       |
| [`logs`](logs.md)             | Fetch the logs of a container                                                 |
| [`manifest`](manifest.md)     | Manage Docker image manifests and manifest lists                              |
| [`network`](network.md)       | Manage networks                                                               |
| [`node`](node.md)             | Manage Swarm nodes                                                            |
| [`pause`](pause.md)           | Pause all processes within one or more containers                             |
| [`plugin`](plugin.md)         | Manage plugins                                                                |
| [`port`](port.md)             | List port mappings or a specific mapping for the container                    |
| [`ps`](ps.md)                 | List containers                                                               |
| [`pull`](pull.md)             | Download an image from a registry                                             |
| [`push`](push.md)             | Upload an image to a registry                                                 |
| [`rename`](rename.md)         | Rename a container                                                            |
| [`restart`](restart.md)       | Restart one or more containers                                                |
| [`rm`](rm.md)                 | Remove one or more containers                                                 |
| [`rmi`](rmi.md)               | Remove one or more images                                                     |
| [`run`](run.md)               | Create and run a new container from an image                                  |
| [`save`](save.md)             | Save one or more images to a tar archive (streamed to STDOUT by default)      |
| [`search`](search.md)         | Search Docker Hub for images                                                  |
| [`secret`](secret.md)         | Manage Swarm secrets                                                          |
| [`service`](service.md)       | Manage Swarm services                                                         |
| [`stack`](stack.md)           | Manage Swarm stacks                                                           |
| [`start`](start.md)           | Start one or more stopped containers                                          |
| [`stats`](stats.md)           | Display a live stream of container(s) resource usage statistics               |
| [`stop`](stop.md)             | Stop one or more running containers                                           |
| [`swarm`](swarm.md)           | Manage Swarm                                                                  |
| [`system`](system.md)         | Manage Docker                                                                 |
| [`tag`](tag.md)               | Create a tag TARGET_IMAGE that refers to SOURCE_IMAGE                         |
| [`top`](top.md)               | Display the running processes of a container                                  |
| [`trust`](trust.md)           | Manage trust on Docker images                                                 |
| [`unpause`](unpause.md)       | Unpause all processes within one or more containers                           |
| [`update`](update.md)         | Update configuration of one or more containers                                |
| [`version`](version.md)       | Show the Docker version information                                           |
| [`volume`](volume.md)         | Manage volumes                                                                |
| [`wait`](wait.md)             | Block until one or more containers stop, then print their exit codes          |


### Options

| Name                             | Type     | Default                  | Description                                                                                                                           |
|:---------------------------------|:---------|:-------------------------|:--------------------------------------------------------------------------------------------------------------------------------------|
| `--config`                       | `string` | `/root/.docker`          | Location of client config files                                                                                                       |
| `-c`, `--context`                | `string` |                          | Name of the context to use to connect to the daemon (overrides DOCKER_HOST env var and default context set with `docker context use`) |
| `-D`, `--debug`                  |          |                          | Enable debug mode                                                                                                                     |
| [`-H`](#host), [`--host`](#host) | `list`   |                          | Daemon socket to connect to                                                                                                           |
| `-l`, `--log-level`              | `string` | `info`                   | Set the logging level (`debug`, `info`, `warn`, `error`, `fatal`)                                                                     |
| `--tls`                          |          |                          | Use TLS; implied by --tlsverify                                                                                                       |
| `--tlscacert`                    | `string` | `/root/.docker/ca.pem`   | Trust certs signed only by this CA                                                                                                    |
| `--tlscert`                      | `string` | `/root/.docker/cert.pem` | Path to TLS certificate file                                                                                                          |
| `--tlskey`                       | `string` | `/root/.docker/key.pem`  | Path to TLS key file                                                                                                                  |
| `--tlsverify`                    |          |                          | Use TLS and verify the remote                                                                                                         |


<!---MARKER_GEN_END-->

## Description

Depending on your Docker system configuration, you may be required to preface
each `docker` command with `sudo`. To avoid having to use `sudo` with the
`docker` command, your system administrator can create a Unix group called
`docker` and add users to it.

For more information about installing Docker or `sudo` configuration, refer to
the [installation](https://docs.docker.com/install/) instructions for your operating system.

## Examples

### <a name="host"></a> Specify daemon host (-H, --host)

You can use the `-H`, `--host` flag to specify a socket to use when you invoke
a `docker` command. You can use the following protocols:

| Scheme                                 | Description               | Example                          |
|----------------------------------------|---------------------------|----------------------------------|
| `unix://[<path>]`                      | Unix socket (Linux only)  | `unix:///var/run/docker.sock`    |
| `tcp://[<IP or host>[:port]]`          | TCP connection            | `tcp://174.17.0.1:2376`          |
| `ssh://[username@]<IP or host>[:port]` | SSH connection            | `ssh://user@192.168.64.5`        |
| `npipe://[<name>]`                     | Named pipe (Windows only) | `npipe:////./pipe/docker_engine` |

If you don't specify the `-H` flag, and you're not using a custom
[context](https://docs.docker.com/engine/context/working-with-contexts),
commands use the following default sockets:

- `unix:///var/run/docker.sock` on macOS and Linux
- `npipe:////./pipe/docker_engine` on Windows

To achieve a similar effect without having to specify the `-H` flag for every
command, you could also [create a context](context_create.md),
or alternatively, use the
[`DOCKER_HOST` environment variable](#environment-variables).

For more information about the `-H` flag, see
[Daemon socket option](dockerd.md#daemon-socket-option).

#### Using TCP sockets

The following example shows how to invoke `docker ps` over TCP, to a remote
daemon with IP address `174.17.0.1`, listening on port `2376`:

```console
$ docker -H tcp://174.17.0.1:2376 ps
```

> **Note**
>
> By convention, the Docker daemon uses port `2376` for secure TLS connections,
> and port `2375` for insecure, non-TLS connections.

#### Using SSH sockets

When you use SSH invoke a command on a remote daemon, the request gets forwarded
to the `/var/run/docker.sock` Unix socket on the SSH host.

```console
$ docker -H ssh://user@192.168.64.5 ps
```

### Display help text

To list the help on any command just execute the command, followed by the
`--help` option.

```console
$ docker run --help

Usage: docker run [OPTIONS] IMAGE [COMMAND] [ARG...]

Create and run a new container from an image

Options:
      --add-host value             Add a custom host-to-IP mapping (host:ip) (default [])
  -a, --attach value               Attach to STDIN, STDOUT or STDERR (default [])
<...>
```

