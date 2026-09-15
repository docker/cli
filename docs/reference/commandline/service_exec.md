# service exec

<!---MARKER_GEN_START-->
Execute a command in a running task of a service, on whichever node it runs

### Options

| Name                  | Type          | Default | Description                                                |
|:----------------------|:--------------|:--------|:-----------------------------------------------------------|
| `--detach-keys`       | `string`      |         | Override the key sequence for detaching a container        |
| `-e`, `--env`         | `list`        |         | Set environment variables                                  |
| `-i`, `--interactive` | `bool`        |         | Keep STDIN open even if not attached                       |
| `--ssh-option`        | `stringSlice` |         | Additional flags passed to ssh (e.g. `-J bastion`)         |
| `--ssh-user`          | `string`      |         | Username for the SSH connection to the node                |
| `--task-id`           | `string`      |         | Exec into a specific task instead of the first running one |
| `-t`, `--tty`         | `bool`        |         | Allocate a pseudo-TTY                                      |
| `-u`, `--user`        | `string`      |         | Username or UID (format: `<name\|uid>[:<group\|gid>]`)     |
| `-w`, `--workdir`     | `string`      |         | Working directory inside the container                     |


<!---MARKER_GEN_END-->

## Description

Executes a command in a running task of a service, wherever that task is
currently scheduled, without having to figure out the node and container
yourself.

The command resolves the service to a running task (the first running one,
or the one given with `--task-id`). If the task runs on the node the client
is already connected to, this behaves exactly like `docker exec`. If the
task runs on another node, the client opens an SSH connection to that node
(the same mechanism as `DOCKER_HOST=ssh://`, using `docker system
dial-stdio` on the remote side) and runs the exec through it, so
interactive sessions, TTY allocation, and exit-code propagation work the
same as a local `docker exec`.

Reaching a remote node requires:

- SSH access to the node, as the current user or the one given with
  `--ssh-user` (key-based authentication; the connection is established
  with the local `ssh` binary, so `~/.ssh/config` is honored).
- The `docker` CLI in the `PATH` of the remote user, with access to the
  local engine socket.

The address used to reach the node is the address advertised in the node's
status (`docker node inspect --format '{{ .Status.Addr }}'`), falling back
to the node hostname when unspecified. Use `--ssh-option` to pass extra
flags to ssh (for example `--ssh-option "-J bastion"` to go through a jump
host). Options taking a value are best passed in `-o Key=value` form (for
example `--ssh-option "-oIdentityFile=~/.ssh/swarm_key"`), matching the
behavior of `DOCKER_HOST=ssh://` connections.

> [!NOTE]
> This command has to be run targeting a manager node.

## Examples

Open a shell in the (single) task of a service, wherever it runs:

```console
$ docker service exec -it myapp_web sh
executing on node worker-2 (kf1r2caqpuivn5fq1o0dj1c1s)
/ #
```

Run a command in a specific task of a service:

```console
$ docker service ps myapp_web --format '{{ .ID }} {{ .Node }}'
wxn1w1twpr5f worker-2
u1o8lhbmja0z worker-3
$ docker service exec --task-id u1o8lhbmja0z myapp_web cat /etc/hostname
executing on node worker-3 (bpn0umb69befk32u9k1hfrf0k)
1742f9c6f1e4
```

## Related commands

* [service inspect](service_inspect.md)
* [service logs](service_logs.md)
* [service ls](service_ls.md)
* [service ps](service_ps.md)
