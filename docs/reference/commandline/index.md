---
title: "Docker commands"
description: "Docker's CLI command description and usage"
keywords: "Docker, Docker documentation, CLI, command line"
identifier: "smn_cli_guide"
---

# The Docker commands

This section contains reference information on using Docker's command line
client. Each command has a reference page along with samples. If you are
unfamiliar with the command line, you should start by reading about how to [Use
the Docker command line](https://docs.docker.com/engine/reference/commandline/cli/).

You start the Docker daemon with the command line. How you start the daemon
affects your Docker containers. For that reason you should also make sure to
read the [`dockerd`](https://docs.docker.com/reference/cli/dockerd/) reference page.

## Commands by object

### Docker management commands

| Command                           | Description                                          |
| :-------------------------------- | :--------------------------------------------------- |
| [dockerd](../dockerd.md)          | Launch the Docker daemon                             |
| [inspect](inspect.md)             | Return low-level information on a container or image |
| [system events](system_events.md) | Get real-time events from the server                 |
| [system info](system_info.md)     | Display system-wide information                      |
| [version](version.md)             | Show the Docker version information                  |

### Image commands

| Command                           | Description                                                     |
| :-------------------------------- | :-------------------------------------------------------------- |
| [image build](image_build.md)     | Build an image from a Dockerfile                                |
| [image commit](image_commit.md)   | Create a new image from a container's changes                   |
| [image history](image_history.md) | Show the history of an image                                    |
| [image import](image_import.md)   | Import the contents from a tarball to create a filesystem image |
| [image load](image_load.md)       | Load an image from a tar archive or STDIN                       |
| [image ls](image_ls.md)           | List images                                                     |
| [image prune](image_prune.md)     | Remove unused images                                            |
| [image rm](image_rm.md)           | Remove one or more images                                       |
| [image save](image_save.md)       | Save images to a tar archive                                    |
| [image tag](image_tag.md)         | Tag an image into a repository                                  |

### Container commands

| Command                                   | Description                                                     |
| :---------------------------------------- | :-------------------------------------------------------------- |
| [container attach](container_attach.md)   | Attach to a running container                                   |
| [container cp](container_cp.md)           | Copy files/folders from a container to a HOSTDIR or to STDOUT   |
| [container create](container_create.md)   | Create a new container                                          |
| [container diff](container_diff.md)       | Inspect changes on a container's filesystem                     |
| [container exec](container_exec.md)       | Execute a command in a running container                        |
| [container export](container_export.md)   | Export a container's filesystem as a tar archive                |
| [container kill](container_kill.md)       | Kill a running container                                        |
| [container logs](container_logs.md)       | Fetch the logs of a container                                   |
| [container ls](container_ls.md)           | List containers                                                 |
| [container pause](container_pause.md)     | Pause all processes within a container                          |
| [container port](container_port.md)       | List port mappings or a specific mapping for the container      |
| [container prune](container_prune.md)     | Remove all stopped containers                                   |
| [container rename](container_rename.md)   | Rename a container                                              |
| [container restart](container_restart.md) | Restart a running container                                     |
| [container rm](container_rm.md)           | Remove one or more containers                                   |
| [container run](container_run.md)         | Create and run a new container from an image                    |
| [container start](container_start.md)     | Start one or more stopped containers                            |
| [container stats](container_stats.md)     | Display a live stream of container(s) resource usage statistics |
| [container stop](container_stop.md)       | Stop a running container                                        |
| [container top](container_top.md)         | Display the running processes of a container                    |
| [container unpause](container_unpause.md) | Unpause all processes within a container                        |
| [container update](container_update.md)   | Update configuration of one or more containers                  |
| [container wait](container_wait.md)       | Block until a container stops, then print its exit code         |

### Hub and registry commands

| Command             | Description                       |
| :------------------ | :-------------------------------- |
| [login](login.md)   | Log in to a registry              |
| [logout](logout.md) | Log out from a registry           |
| [pull](pull.md)     | Download an image from a registry |
| [push](push.md)     | Upload an image to a registry     |
| [search](search.md) | Search Docker Hub for images      |

### Network and connectivity commands

| Command                                     | Description                                            |
| :------------------------------------------ | :----------------------------------------------------- |
| [network connect](network_connect.md)       | Connect a container to a network                       |
| [network create](network_create.md)         | Create a new network                                   |
| [network disconnect](network_disconnect.md) | Disconnect a container from a network                  |
| [network inspect](network_inspect.md)       | Display information about a network                    |
| [network ls](network_ls.md)                 | Lists all the networks the Engine `daemon` knows about |
| [network prune](network_prune.md)           | Remove all unused networks                             |
| [network rm](network_rm.md)                 | Removes one or more networks                           |

### Shared data volume commands

| Command                             | Description                                                      |
| :---------------------------------- | :--------------------------------------------------------------- |
| [volume create](volume_create.md)   | Creates a new volume where containers can consume and store data |
| [volume inspect](volume_inspect.md) | Display information about a volume                               |
| [volume ls](volume_ls.md)           | Lists all the volumes Docker knows about                         |
| [volume prune](volume_prune.md)     | Remove unused local volumes                                      |
| [volume rm](volume_rm.md)           | Remove one or more volumes                                       |

### Swarm node commands

| Command                         | Description                                                   |
| :------------------------------ | :------------------------------------------------------------ |
| [node demote](node_demote.md)   | Demotes an existing manager so that it is no longer a manager |
| [node inspect](node_inspect.md) | Inspect a node in the swarm                                   |
| [node ls](node_ls.md)           | List nodes in the swarm                                       |
| [node promote](node_promote.md) | Promote a node that is pending a promotion to manager         |
| [node ps](node_ps.md)           | List tasks running on one or more nodes                       |
| [node rm](node_rm.md)           | Remove one or more nodes from the swarm                       |
| [node update](node_update.md)   | Update attributes for a node                                  |

### Swarm management commands

| Command                                 | Description                                   |
| :-------------------------------------- | :-------------------------------------------- |
| [swarm init](swarm_init.md)             | Initialize a swarm                            |
| [swarm join-token](swarm_join-token.md) | Display or rotate join tokens                 |
| [swarm join](swarm_join.md)             | Join a swarm as a manager node or worker node |
| [swarm leave](swarm_leave.md)           | Remove the current node from the swarm        |
| [swarm unlock-key](swarm_unlock-key.md) | Manage the unlock key                         |
| [swarm unlock](swarm_unlock.md)         | Unlock swarm                                  |
| [swarm update](swarm_update.md)         | Update attributes of a swarm                  |

### Swarm service commands

| Command                               | Description                                                     |
| :------------------------------------ | :-------------------------------------------------------------- |
| [service create](service_create.md)   | Create a new service                                            |
| [service inspect](service_inspect.md) | Inspect a service                                               |
| [service logs](service_logs.md)       | Fetch the logs of a service or task                             |
| [service ls](service_ls.md)           | List services in the swarm                                      |
| [service ps](service_ps.md)           | List the tasks of a service                                     |
| [service rm](service_rm.md)           | Remove a service from the swarm                                 |
| [service scale](service_scale.md)     | Set the number of replicas for the desired state of the service |
| [service update](service_update.md)   | Update the attributes of a service                              |

### Swarm secret commands

| Command                              | Description                                     |
| :----------------------------------- | :---------------------------------------------- |
| [secret create](secret_create.md)    | Create a secret from a file or STDIN as content |
| [secret inspect](service_inspect.md) | Inspect the specified secret                    |
| [secret ls](secret_ls.md)            | List secrets in the swarm                       |
| [secret rm](secret_rm.md)            | Remove the specified secrets from the swarm     |

### Swarm stack commands

| Command                             | Description                                             |
| :---------------------------------- | :------------------------------------------------------ |
| [stack config](stack_config.md)     | Output the Compose file after merges and interpolations |
| [stack deploy](stack_deploy.md)     | Deploy a new stack or update an existing stack          |
| [stack ls](stack_ls.md)             | List stacks in the swarm                                |
| [stack ps](stack_ps.md)             | List the tasks in the stack                             |
| [stack rm](stack_rm.md)             | Remove the stack from the swarm                         |
| [stack services](stack_services.md) | List the services in the stack                          |

### Plugin commands

| Command                             | Description                                     |
| :---------------------------------- | :---------------------------------------------- |
| [plugin create](plugin_create.md)   | Create a plugin from a rootfs and configuration |
| [plugin disable](plugin_disable.md) | Disable a plugin                                |
| [plugin enable](plugin_enable.md)   | Enable a plugin                                 |
| [plugin inspect](plugin_inspect.md) | Display detailed information on a plugin        |
| [plugin install](plugin_install.md) | Install a plugin                                |
| [plugin ls](plugin_ls.md)           | List plugins                                    |
| [plugin push](plugin_push.md)       | Push a plugin to a registry                     |
| [plugin rm](plugin_rm.md)           | Remove a plugin                                 |
| [plugin set](plugin_set.md)         | Change settings for a plugin                    |

### Context commands

| Command                               | Description                    |
| :------------------------------------ | :----------------------------- |
| [context create](context_create.md)   | Create a context               |
| [context export](context_export.md)   | Export a context               |
| [context import](context_import.md)   | Import a context               |
| [context inspect](context_inspect.md) | Inspect one or more contexts   |
| [context ls](context_ls.md)           | List contexts                  |
| [context rm](context_rm.md)           | Remove one or more contexts    |
| [context update](context_update.md)   | Update a context               |
| [context use](context_use.md)         | Set the current docker context |
