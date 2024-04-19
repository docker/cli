# secret ls

<!---MARKER_GEN_START-->
List secrets

### Aliases

`docker secret ls`, `docker secret list`

### Options

| Name                                   | Type     | Default | Description                                                                                                                                                                                                                                                                                                                                                                                                                          |
|:---------------------------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`-f`](#filter), [`--filter`](#filter) | `filter` |         | Filter output based on conditions provided                                                                                                                                                                                                                                                                                                                                                                                           |
| [`--format`](#format)                  | `string` |         | Format output using a custom template:<br>'table':            Print output in table format with column headers (default)<br>'table TEMPLATE':   Print output in table format using the given Go template<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |
| `-q`, `--quiet`                        |          |         | Only display IDs                                                                                                                                                                                                                                                                                                                                                                                                                     |


<!---MARKER_GEN_END-->

## Description

Run this command on a manager node to list the secrets in the swarm.

For detailed information about using secrets, refer to [manage sensitive data with Docker secrets](https://docs.docker.com/engine/swarm/secrets/).

> **Note**
>
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

```console
$ docker secret ls

ID                          NAME                        CREATED             UPDATED
6697bflskwj1998km1gnnjr38   q5s5570vtvnimefos1fyeo2u2   6 weeks ago         6 weeks ago
9u9hk4br2ej0wgngkga6rp4hq   my_secret                   5 weeks ago         5 weeks ago
mem02h8n73mybpgqjf0kfi1n0   test_secret                 3 seconds ago       3 seconds ago
```

### <a name="filter"></a> Filtering (--filter)

The filtering flag (`-f` or `--filter`) format is a `key=value` pair. If there is more
than one filter, then pass multiple flags (e.g., `--filter "foo=bar" --filter "bif=baz"`).

The currently supported filters are:

- [id](#id) (secret's ID)
- [label](#label) (`label=<key>` or `label=<key>=<value>`)
- [name](#name) (secret's name)

#### id

The `id` filter matches all or prefix of a secret's id.

```console
$ docker secret ls -f "id=6697bflskwj1998km1gnnjr38"

ID                          NAME                        CREATED             UPDATED
6697bflskwj1998km1gnnjr38   q5s5570vtvnimefos1fyeo2u2   6 weeks ago         6 weeks ago
```

#### label

The `label` filter matches secrets based on the presence of a `label` alone or
a `label` and a value.

The following filter matches all secrets with a `project` label regardless of
its value:

```console
$ docker secret ls --filter label=project

ID                          NAME                        CREATED             UPDATED
mem02h8n73mybpgqjf0kfi1n0   test_secret                 About an hour ago   About an hour ago
```

The following filter matches only services with the `project` label with the
`project-a` value.

```console
$ docker service ls --filter label=project=test

ID                          NAME                        CREATED             UPDATED
mem02h8n73mybpgqjf0kfi1n0   test_secret                 About an hour ago   About an hour ago
```

#### name

The `name` filter matches on all or prefix of a secret's name.

The following filter matches secret with a name containing a prefix of `test`.

```console
$ docker secret ls --filter name=test_secret

ID                          NAME                        CREATED             UPDATED
mem02h8n73mybpgqjf0kfi1n0   test_secret                 About an hour ago   About an hour ago
```

### <a name="format"></a> Format the output (--format)

The formatting option (`--format`) pretty prints secrets output
using a Go template.

Valid placeholders for the Go template are listed below:

| Placeholder  | Description                                                                          |
|--------------|--------------------------------------------------------------------------------------|
| `.ID`        | Secret ID                                                                            |
| `.Name`      | Secret name                                                                          |
| `.CreatedAt` | Time when the secret was created                                                     |
| `.UpdatedAt` | Time when the secret was updated                                                     |
| `.Labels`    | All labels assigned to the secret                                                    |
| `.Label`     | Value of a specific label for this secret. For example `{{.Label "secret.ssh.key"}}` |

When using the `--format` option, the `secret ls` command will either
output the data exactly as the template declares or, when using the
`table` directive, will include column headers as well.

The following example uses a template without headers and outputs the
`ID` and `Name` entries separated by a colon (`:`) for all images:

```console
$ docker secret ls --format "{{.ID}}: {{.Name}}"

77af4d6b9913: secret-1
b6fa739cedf5: secret-2
78a85c484f71: secret-3
```

To list all secrets with their name and created date in a table format you
can use:

```console
$ docker secret ls --format "table {{.ID}}\t{{.Name}}\t{{.CreatedAt}}"

ID                  NAME                      CREATED
77af4d6b9913        secret-1                  5 minutes ago
b6fa739cedf5        secret-2                  3 hours ago
78a85c484f71        secret-3                  10 days ago
```

To list all secrets in JSON format, use the `json` directive:
```console
$ docker secret ls --format json
{"CreatedAt":"28 seconds ago","Driver":"","ID":"4y7hvwrt1u8e9uxh5ygqj7mzc","Labels":"","Name":"mysecret","UpdatedAt":"28 seconds ago"}
```

## Related commands

* [secret create](secret_create.md)
* [secret inspect](secret_inspect.md)
* [secret rm](secret_rm.md)
