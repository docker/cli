# docker history

<!---MARKER_GEN_START-->
Show the history of an image

### Aliases

`docker image history`, `docker history`

### Options

| Name            | Type     | Default | Description                                                                                                                                                                                                                                                                                                                                                                                                                          |
|:----------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--format`      | `string` |         | Format output using a custom template:<br>'table':            Print output in table format with column headers (default)<br>'table TEMPLATE':   Print output in table format using the given Go template<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |
| `-H`, `--human` | `bool`   | `true`  | Print sizes and dates in human readable format                                                                                                                                                                                                                                                                                                                                                                                       |
| `--no-trunc`    | `bool`   |         | Don't truncate output                                                                                                                                                                                                                                                                                                                                                                                                                |
| `--platform`    | `string` |         | Show history for the given platform. Formatted as `os[/arch[/variant]]` (e.g., `linux/amd64`)                                                                                                                                                                                                                                                                                                                                        |
| `-q`, `--quiet` | `bool`   |         | Only show image IDs                                                                                                                                                                                                                                                                                                                                                                                                                  |


<!---MARKER_GEN_END-->

