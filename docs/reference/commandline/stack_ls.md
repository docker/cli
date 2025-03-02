# stack ls

<!---MARKER_GEN_START-->
List stacks

### Aliases

`docker stack ls`, `docker stack list`

### Options

| Name                  | Type     | Default | Description                                                                                                                                                                                                                                                                                                                                                                                                                          |
|:----------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`--format`](#format) | `string` |         | Format output using a custom template:<br>'table':            Print output in table format with column headers (default)<br>'table TEMPLATE':   Print output in table format using the given Go template<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |


<!---MARKER_GEN_END-->

## Description

Lists the stacks.

> [!NOTE]
> This is a cluster management command, and must be executed on a swarm
> manager node. To learn about managers and workers, refer to the
> [Swarm mode section](https://docs.docker.com/engine/swarm/) in the
> documentation.

## Examples

The following command shows all stacks and some additional information:

```console
$ docker stack ls

ID                 SERVICES            ORCHESTRATOR
myapp              2                   Kubernetes
vossibility-stack  6                   Swarm
```

### <a name="format"></a> Format the output (--format)

The formatting option (`--format`) pretty-prints stacks using a Go template.

Valid placeholders for the Go template are listed below:

| Placeholder     | Description        |
|-----------------|--------------------|
| `.Name`         | Stack name         |
| `.Services`     | Number of services |
| `.Orchestrator` | Orchestrator name  |
| `.Namespace`    | Namespace          |

When using the `--format` option, the `stack ls` command either outputs
the data exactly as the template declares or, when using the
`table` directive, includes column headers as well.

The following example uses a template without headers and outputs the
`Name` and `Services` entries separated by a colon (`:`) for all stacks:

```console
$ docker stack ls --format "{{.Name}}: {{.Services}}"
web-server: 1
web-cache: 4
```

To list all stacks in JSON format, use the `json` directive:

```console
$ docker stack ls --format json
{"Name":"myapp","Namespace":"","Orchestrator":"Swarm","Services":"3"}
```

## Related commands

* [stack deploy](stack_deploy.md)
* [stack ps](stack_ps.md)
* [stack rm](stack_rm.md)
* [stack services](stack_services.md)
