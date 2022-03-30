---
title: "node ls"
description: "The node ls command description and usage"
keywords: "node, list"
---

# node ls

```markdown
Usage:  docker node ls [OPTIONS]

List nodes in the swarm

Aliases:
  ls, list

Options:
  -f, --filter filter   Filter output based on conditions provided
      --format string   Format output using a custom template:
                        'table':            Print output in table format with column headers (default)
                        'table TEMPLATE':   Print output in table format using the given Go template
                        'json':             Print in JSON format
                        'TEMPLATE':         Print output using the given Go template.
                        Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates
      --help            Print usage
  -q, --quiet           Only display IDs
```

## Description

Lists all the nodes that the Docker Swarm manager knows about. You can filter
using the `-f` or `--filter` flag. Refer to the [filtering](#filtering) section
for more information about available filter options.

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

```console
$ docker node ls

ID                           HOSTNAME        STATUS  AVAILABILITY  MANAGER STATUS
1bcef6utixb0l0ca7gxuivsj0    swarm-worker2   Ready   Active
38ciaotwjuritcdtn9npbnkuz    swarm-worker1   Ready   Active
e216jshn25ckzbvmwlnh5jr3g *  swarm-manager1  Ready   Active        Leader
```

> **Note**
>
> In the above example output, there is a hidden column of `.Self` that indicates
> if the node is the same node as the current docker daemon. A `*` (e.g.,
> `e216jshn25ckzbvmwlnh5jr3g *`) means this node is the current docker daemon.


### Filtering

The filtering flag (`-f` or `--filter`) format is of "key=value". If there is more
than one filter, then pass multiple flags (e.g., `--filter "foo=bar" --filter "bif=baz"`)

The currently supported filters are:

* [id](#id)
* [label](#label)
* [node.label](#nodelabel)
* [membership](#membership)
* [name](#name)
* [role](#role)

#### id

The `id` filter matches all or part of a node's id.

```console
$ docker node ls -f id=1

ID                         HOSTNAME       STATUS  AVAILABILITY  MANAGER STATUS
1bcef6utixb0l0ca7gxuivsj0  swarm-worker2  Ready   Active
```

#### label

The `label` filter matches nodes based on engine labels and on the presence of a
`label` alone or a `label` and a value. Engine labels are configured in
the [daemon configuration](dockerd.md#daemon-configuration-file). To filter on
Swarm `node` labels, use [`node.label` instead](#nodelabel).

The following filter matches nodes with the `foo` label regardless of its value.

```console
$ docker node ls -f "label=foo"

ID                         HOSTNAME       STATUS  AVAILABILITY  MANAGER STATUS
1bcef6utixb0l0ca7gxuivsj0  swarm-worker2  Ready   Active
```

#### node.label

The `node.label` filter matches nodes based on node labels and on the presence
of a `node.label` alone or a `node.label` and a value.

The following filter updates nodes to have a `region` node label:

```console
$ docker node update --label-add region=region-a swarm-test-01
$ docker node update --label-add region=region-a swarm-test-02
$ docker node update --label-add region=region-b swarm-test-03
$ docker node update --label-add region=region-b swarm-test-04
```

Show all nodes that have a `region` node label set:

```console
$ docker node ls --filter node.label=region

ID                            HOSTNAME        STATUS    AVAILABILITY   MANAGER STATUS   ENGINE VERSION
yg550ettvsjn6g6t840iaiwgb *   swarm-test-01   Ready     Active         Leader           20.10.2
2lm9w9kbepgvkzkkeyku40e65     swarm-test-02   Ready     Active         Reachable        20.10.2
hc0pu7ntc7s4uvj4pv7z7pz15     swarm-test-03   Ready     Active         Reachable        20.10.2
n41b2cijmhifxxvz56vwrs12q     swarm-test-04   Ready     Active                          20.10.2
```

Show all nodes that have a `region` node label, with value `region-a`:

```console
$ docker node ls --filter node.label=region=region-a

ID                            HOSTNAME        STATUS    AVAILABILITY   MANAGER STATUS   ENGINE VERSION
yg550ettvsjn6g6t840iaiwgb *   swarm-test-01   Ready     Active         Leader           20.10.2
2lm9w9kbepgvkzkkeyku40e65     swarm-test-02   Ready     Active         Reachable        20.10.2
```

#### membership

The `membership` filter matches nodes based on the presence of a `membership` and a value
`accepted` or `pending`.

The following filter matches nodes with the `membership` of `accepted`.

```console
$ docker node ls -f "membership=accepted"

ID                           HOSTNAME        STATUS  AVAILABILITY  MANAGER STATUS
1bcef6utixb0l0ca7gxuivsj0    swarm-worker2   Ready   Active
38ciaotwjuritcdtn9npbnkuz    swarm-worker1   Ready   Active
```

#### name

The `name` filter matches on all or part of a node hostname.

The following filter matches the nodes with a name equal to `swarm-master` string.

```console
$ docker node ls -f name=swarm-manager1

ID                           HOSTNAME        STATUS  AVAILABILITY  MANAGER STATUS
e216jshn25ckzbvmwlnh5jr3g *  swarm-manager1  Ready   Active        Leader
```

#### role

The `role` filter matches nodes based on the presence of a `role` and a value `worker` or `manager`.

The following filter matches nodes with the `manager` role.

```console
$ docker node ls -f "role=manager"

ID                           HOSTNAME        STATUS  AVAILABILITY  MANAGER STATUS
e216jshn25ckzbvmwlnh5jr3g *  swarm-manager1  Ready   Active        Leader
```

### Formatting

The formatting options (`--format`) pretty-prints nodes output
using a Go template.

Valid placeholders for the Go template are listed below:

| Placeholder      | Description                                                                                           |
|------------------|-------------------------------------------------------------------------------------------------------|
| `.ID`            | Node ID                                                                                               |
| `.Self`          | Node of the daemon (`true/false`, `true`indicates that the node is the same as current docker daemon) |
| `.Hostname`      | Node hostname                                                                                         |
| `.Status`        | Node status                                                                                           |
| `.Availability`  | Node availability ("active", "pause", or "drain")                                                     |
| `.ManagerStatus` | Manager status of the node                                                                            |
| `.TLSStatus`     | TLS status of the node ("Ready", or "Needs Rotation" has TLS certificate signed by an old CA)         |
| `.EngineVersion` | Engine version                                                                                        |

When using the `--format` option, the `node ls` command will either
output the data exactly as the template declares or, when using the
`table` directive, includes column headers as well.

The following example uses a template without headers and outputs the
`ID`, `Hostname`, and `TLS Status` entries separated by a colon (`:`) for all
nodes:

```console
$ docker node ls --format "{{.ID}}: {{.Hostname}} {{.TLSStatus}}"

e216jshn25ckzbvmwlnh5jr3g: swarm-manager1 Ready
35o6tiywb700jesrt3dmllaza: swarm-worker1 Needs Rotation
```

To list all nodes in JSON format, use the `json` directive:
```console
$ docker node ls --format json
{"Availability":"Active","EngineVersion":"20.10.5","Hostname":"docker-desktop","ID":"k8f4w7qtzpj5sqzclcqafw35g","ManagerStatus":"Leader","Self":true,"Status":"Ready","TLSStatus":"Ready"}
```

## Related commands

* [node demote](node_demote.md)
* [node inspect](node_inspect.md)
* [node promote](node_promote.md)
* [node ps](node_ps.md)
* [node rm](node_rm.md)
* [node update](node_update.md)
