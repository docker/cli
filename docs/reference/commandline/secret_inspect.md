# secret inspect

<!---MARKER_GEN_START-->
Display detailed information on one or more secrets

### Options

| Name                                   | Type     | Default | Description                                                                                                                                                                                                                                                        |
|:---------------------------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`-f`](#format), [`--format`](#format) | `string` |         | Format output using a custom template:<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |
| `--pretty`                             |          |         | Print the information in a human friendly format                                                                                                                                                                                                                   |


<!---MARKER_GEN_END-->

## Description

Inspects the specified secret.

By default, this renders all results in a JSON array. If a format is specified,
the given template will be executed for each result.

Go's [text/template](https://pkg.go.dev/text/template) package
describes all the details of the format.

For detailed information about using secrets, refer to [manage sensitive data with Docker secrets](https://docs.docker.com/engine/swarm/secrets/).

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

### Inspect a secret by name or ID

You can inspect a secret, either by its name or ID.

For example, given the following secret:

```console
$ docker secret ls

ID                          NAME                CREATED             UPDATED
eo7jnzguqgtpdah3cm5srfb97   my_secret           3 minutes ago       3 minutes ago
```

```console
$ docker secret inspect secret.json
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
      "Name": "my_secret",
      "Labels": {
        "env": "dev",
        "rev": "20170324"
      }
    }
  }
]
```

### <a name="format"></a> Format the output (--format)

You can use the `--format` option to obtain specific information about a
secret. The following example command outputs the creation time of the
secret.

```console
$ docker secret inspect --format='{{.CreatedAt}}' eo7jnzguqgtpdah3cm5srfb97

2017-03-24 08:15:09.735271783 +0000 UTC
```


## Related commands

* [secret create](secret_create.md)
* [secret ls](secret_ls.md)
* [secret rm](secret_rm.md)
