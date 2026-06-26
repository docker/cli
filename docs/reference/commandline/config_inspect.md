# config inspect

<!---MARKER_GEN_START-->
Display detailed information on one or more configs

### Options

| Name                                   | Type     | Default | Description                                                                                                                                                                                                                                                        |
|:---------------------------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`-f`](#format), [`--format`](#format) | `string` |         | Format output using a custom template:<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |
| `--pretty`                             | `bool`   |         | Print the information in a human friendly format                                                                                                                                                                                                                   |


<!---MARKER_GEN_END-->

## Description

Inspects the specified config.

By default, this renders all results in a JSON array. If a format is specified,
the given template will be executed for each result.

Go's [text/template](https://pkg.go.dev/text/template) package
describes all the details of the format.

For detailed information about using configs, refer to [store configuration data using Docker Configs](https://docs.docker.com/engine/swarm/configs/).

> [!NOTE]
> This is a cluster management command, and must be executed on a Swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### Inspect a config by name or ID

You can inspect a config, either by its *name*, or *ID*

For example, given the following config:

```console
$ docker config ls

ID                          NAME                CREATED             UPDATED
eo7jnzguqgtpdah3cm5srfb97   my_config           3 minutes ago       3 minutes ago
```

```console
$ docker config inspect config.json
```

The output is in JSON format, for example:

```json
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

### <a name="format"></a> Format the output (--format)

You can use the --format option to obtain specific information about a
config. The following example command outputs the creation time of the
config.

```console
$ docker config inspect --format='{{.CreatedAt}}' eo7jnzguqgtpdah3cm5srfb97

2017-03-24 08:15:09.735271783 +0000 UTC
```

## Related commands

* [config create](config_create.md)
* [config ls](config_ls.md)
* [config rm](config_rm.md)
