# docker

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

| Name                | Type     | Default                  | Description                                                                                                                           |
|:--------------------|:---------|:-------------------------|:--------------------------------------------------------------------------------------------------------------------------------------|
| `--config`          | `string` | `/root/.docker`          | Location of client config files                                                                                                       |
| `-c`, `--context`   | `string` |                          | Name of the context to use to connect to the daemon (overrides DOCKER_HOST env var and default context set with `docker context use`) |
| `-D`, `--debug`     |          |                          | Enable debug mode                                                                                                                     |
| `-H`, `--host`      | `list`   |                          | Daemon socket to connect to                                                                                                           |
| `-l`, `--log-level` | `string` | `info`                   | Set the logging level (`debug`, `info`, `warn`, `error`, `fatal`)                                                                     |
| `--tls`             |          |                          | Use TLS; implied by --tlsverify                                                                                                       |
| `--tlscacert`       | `string` | `/root/.docker/ca.pem`   | Trust certs signed only by this CA                                                                                                    |
| `--tlscert`         | `string` | `/root/.docker/cert.pem` | Path to TLS certificate file                                                                                                          |
| `--tlskey`          | `string` | `/root/.docker/key.pem`  | Path to TLS key file                                                                                                                  |
| `--tlsverify`       |          |                          | Use TLS and verify the remote                                                                                                         |


<!---MARKER_GEN_END-->

