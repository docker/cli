# update

<!---MARKER_GEN_START-->
Update a node

### Options

| Name                        | Type     | Default | Description                                           |
|:----------------------------|:---------|:--------|:------------------------------------------------------|
| `--availability`            | `string` |         | Availability of the node (`active`, `pause`, `drain`) |
| [`--label-add`](#label-add) | `list`   |         | Add or update a node label (`key=value`)              |
| `--label-rm`                | `list`   |         | Remove a node label if exists                         |
| `--role`                    | `string` |         | Role of the node (`worker`, `manager`)                |


<!---MARKER_GEN_END-->

## Description

Update metadata about a node, such as its availability, labels, or roles.

> [!NOTE]
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### <a name="label-add"></a> Add label metadata to a node (--label-add)

Add metadata to a swarm node using node labels. You can specify a node label as
a key with an empty value:

``` bash
$ docker node update --label-add foo worker1
```

To add multiple labels to a node, pass the `--label-add` flag for each label:

```console
$ docker node update --label-add foo --label-add bar worker1
```

When you [create a service](service_create.md),
you can use node labels as a constraint. A constraint limits the nodes where the
scheduler deploys tasks for a service.

For example, to add a `type` label to identify nodes where the scheduler should
deploy message queue service tasks:

``` bash
$ docker node update --label-add type=queue worker1
```

The labels you set for nodes using `docker node update` apply only to the node
entity within the swarm. Do not confuse them with the docker daemon labels for
[dockerd](https://docs.docker.com/reference/cli/dockerd/).

For more information about labels, refer to [apply custom
metadata](https://docs.docker.com/engine/userguide/labels-custom-metadata/).

## Related commands

* [node demote](node_demote.md)
* [node inspect](node_inspect.md)
* [node ls](node_ls.md)
* [node promote](node_promote.md)
* [node ps](node_ps.md)
* [node rm](node_rm.md)
