# node ps

<!---MARKER_GEN_START-->
List tasks running on one or more nodes, defaults to current node

### Options

| Name                                   | Type     | Default | Description                                |
|:---------------------------------------|:---------|:--------|:-------------------------------------------|
| [`-f`](#filter), [`--filter`](#filter) | `filter` |         | Filter output based on conditions provided |
| [`--format`](#format)                  | `string` |         | Pretty-print tasks using a Go template     |
| `--no-resolve`                         | `bool`   |         | Do not map IDs to Names                    |
| `--no-trunc`                           | `bool`   |         | Do not truncate output                     |
| `-q`, `--quiet`                        | `bool`   |         | Only display task IDs                      |


<!---MARKER_GEN_END-->

## Description

Lists all the tasks on a Node that Docker knows about. You can filter using the
`-f` or `--filter` flag. Refer to the [filtering](#filter) section for more
information about available filter options.

> [!NOTE]
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

```console
$ docker node ps swarm-manager1

NAME                                IMAGE        NODE            DESIRED STATE  CURRENT STATE
redis.1.7q92v0nr1hcgts2amcjyqg3pq   redis:7.4.1  swarm-manager1  Running        Running 5 hours
redis.6.b465edgho06e318egmgjbqo4o   redis:7.4.1  swarm-manager1  Running        Running 29 seconds
redis.7.bg8c07zzg87di2mufeq51a2qp   redis:7.4.1  swarm-manager1  Running        Running 5 seconds
redis.9.dkkual96p4bb3s6b10r7coxxt   redis:7.4.1  swarm-manager1  Running        Running 5 seconds
redis.10.0tgctg8h8cech4w0k0gwrmr23  redis:7.4.1  swarm-manager1  Running        Running 5 seconds
```

### <a name="filter"></a> Filtering (--filter)

The filtering flag (`-f` or `--filter`) format is of "key=value". If there is
more than one filter, then pass multiple flags (e.g., `--filter "foo=bar"
--filter "bif=baz"`).

The currently supported filters are:

* [name](#name)
* [id](#id)
* [label](#label)
* [desired-state](#desired-state)

#### name

The `name` filter matches on all or part of a task's name.

The following filter matches all tasks with a name containing the `redis` string.

```console
$ docker node ps -f name=redis swarm-manager1

NAME                                IMAGE        NODE            DESIRED STATE  CURRENT STATE
redis.1.7q92v0nr1hcgts2amcjyqg3pq   redis:7.4.1  swarm-manager1  Running        Running 5 hours
redis.6.b465edgho06e318egmgjbqo4o   redis:7.4.1  swarm-manager1  Running        Running 29 seconds
redis.7.bg8c07zzg87di2mufeq51a2qp   redis:7.4.1  swarm-manager1  Running        Running 5 seconds
redis.9.dkkual96p4bb3s6b10r7coxxt   redis:7.4.1  swarm-manager1  Running        Running 5 seconds
redis.10.0tgctg8h8cech4w0k0gwrmr23  redis:7.4.1  swarm-manager1  Running        Running 5 seconds
```

#### id

The `id` filter matches a task's id.

```console
$ docker node ps -f id=bg8c07zzg87di2mufeq51a2qp swarm-manager1

NAME                                IMAGE        NODE            DESIRED STATE  CURRENT STATE
redis.7.bg8c07zzg87di2mufeq51a2qp   redis:7.4.1  swarm-manager1  Running        Running 5 seconds
```

#### label

The `label` filter matches tasks based on the presence of a `label` alone or a `label` and a
value.

The following filter matches tasks with the `usage` label regardless of its value.

```console
$ docker node ps -f "label=usage"

NAME                               IMAGE        NODE            DESIRED STATE  CURRENT STATE
redis.6.b465edgho06e318egmgjbqo4o  redis:7.4.1  swarm-manager1  Running        Running 10 minutes
redis.7.bg8c07zzg87di2mufeq51a2qp  redis:7.4.1  swarm-manager1  Running        Running 9 minutes
```


#### desired-state

The `desired-state` filter can take the values `running`, `shutdown`, or `accepted`.


### <a name="format"></a> Format the output (--format)

The formatting options (`--format`) pretty-prints tasks output
using a Go template.

Valid placeholders for the Go template are listed below:

| Placeholder     | Description                                                      |
|-----------------|------------------------------------------------------------------|
| `.ID`           | Task ID                                                          |
| `.Name`         | Task name                                                        |
| `.Image`        | Task image                                                       |
| `.Node`         | Node ID                                                          |
| `.DesiredState` | Desired state of the task (`running`, `shutdown`, or `accepted`) |
| `.CurrentState` | Current state of the task                                        |
| `.Error`        | Error                                                            |
| `.Ports`        | Task published ports                                             |

When using the `--format` option, the `node ps` command will either
output the data exactly as the template declares or, when using the
`table` directive, includes column headers as well.

The following example uses a template without headers and outputs the
`Name` and `Image` entries separated by a colon (`:`) for all tasks:

```console
$ docker node ps --format "{{.Name}}: {{.Image}}"

top.1: busybox
top.2: busybox
top.3: busybox
```

## Related commands

* [node demote](node_demote.md)
* [node inspect](node_inspect.md)
* [node ls](node_ls.md)
* [node promote](node_promote.md)
* [node rm](node_rm.md)
* [node update](node_update.md)
