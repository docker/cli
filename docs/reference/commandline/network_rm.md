# network rm

<!---MARKER_GEN_START-->
Remove one or more networks

### Aliases

`docker network rm`, `docker network remove`

### Options

| Name            | Type | Default | Description                                |
|:----------------|:-----|:--------|:-------------------------------------------|
| `-f`, `--force` |      |         | Do not error if the network does not exist |


<!---MARKER_GEN_END-->

## Description

Removes one or more networks by name or identifier. To remove a network,
you must first disconnect any containers connected to it.

## Examples

### Remove a network

To remove the network named 'my-network':

```console
$ docker network rm my-network
```

### Remove multiple networks

To delete multiple networks in a single `docker network rm` command, provide
multiple network names or ids. The following example deletes a network with id
`3695c422697f` and a network named `my-network`:

```console
$ docker network rm 3695c422697f my-network
```

When you specify multiple networks, the command attempts to delete each in turn.
If the deletion of one network fails, the command continues to the next on the
list and tries to delete that. The command reports success or failure for each
deletion.

## Related commands

* [network disconnect ](network_disconnect.md)
* [network connect](network_connect.md)
* [network create](network_create.md)
* [network ls](network_ls.md)
* [network inspect](network_inspect.md)
* [network prune](network_prune.md)
* [Understand Docker container networks](https://docs.docker.com/network/)
