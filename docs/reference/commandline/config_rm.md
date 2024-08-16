# config rm

<!---MARKER_GEN_START-->
Remove one or more configs

### Aliases

`docker config rm`, `docker config remove`


<!---MARKER_GEN_END-->

## Description

Removes the specified configs from the Swarm.

For detailed information about using configs, refer to [store configuration data using Docker Configs](https://docs.docker.com/engine/swarm/configs/).

> [!NOTE]
> This is a cluster management command, and must be executed on a Swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

This example removes a config:

```console
$ docker config rm my_config
sapth4csdo5b6wz2p5uimh5xg
```

> [!WARNING]
> This command doesn't ask for confirmation before removing a config.
{ .warning }

## Related commands

* [config create](config_create.md)
* [config inspect](config_inspect.md)
* [config ls](config_ls.md)
