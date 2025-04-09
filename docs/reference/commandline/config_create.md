# config create

<!---MARKER_GEN_START-->
Create a config from a file or STDIN

### Options

| Name                                | Type     | Default | Description     |
|:------------------------------------|:---------|:--------|:----------------|
| [`-l`](#label), [`--label`](#label) | `list`   |         | Config labels   |
| `--template-driver`                 | `string` |         | Template driver |


<!---MARKER_GEN_END-->

## Description

Creates a config using standard input or from a file for the config content.

For detailed information about using configs, refer to [store configuration data using Docker Configs](https://docs.docker.com/engine/swarm/configs/).

> [!NOTE]
> This is a cluster management command, and must be executed on a Swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### Create a config

```console
$ printf <config> | docker config create my_config -

onakdyv307se2tl7nl20anokv

$ docker config ls

ID                          NAME                CREATED             UPDATED
onakdyv307se2tl7nl20anokv   my_config           6 seconds ago       6 seconds ago
```

### Create a config with a file

```console
$ docker config create my_config ./config.json

dg426haahpi5ezmkkj5kyl3sn

$ docker config ls

ID                          NAME                CREATED             UPDATED
dg426haahpi5ezmkkj5kyl3sn   my_config           7 seconds ago       7 seconds ago
```

### <a name="label"></a> Create a config with labels (-l, --label)

```console
$ docker config create \
    --label env=dev \
    --label rev=20170324 \
    my_config ./config.json

eo7jnzguqgtpdah3cm5srfb97
```

```console
$ docker config inspect my_config

[
    {
        "ID": "eo7jnzguqgtpdah3cm5srfb97",
        "Version": {
            "Index": 17
        },
        "CreatedAt": "2017-03-24T08:15:09.735271783Z",
        "UpdatedAt": "2017-03-24T08:15:09.735271783Z",
        "Spec": {
            "Name": "my_config",
            "Labels": {
                "env": "dev",
                "rev": "20170324"
            },
            "Data": "aGVsbG8K"
        }
    }
]
```

## Related commands

* [config inspect](config_inspect.md)
* [config ls](config_ls.md)
* [config rm](config_rm.md)
