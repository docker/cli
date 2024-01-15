# docker ps

<!---MARKER_GEN_START-->
List containers

### Aliases

`docker container ls`, `docker container list`, `docker container ps`, `docker ps`

### Options

| Name             | Type     | Default | Description                                                                                                                                                                                                                                                                                                                                                                                                                          |
|:-----------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-a`, `--all`    |          |         | Show all containers (default shows just running)                                                                                                                                                                                                                                                                                                                                                                                     |
| `-f`, `--filter` | `filter` |         | Filter output based on conditions provided                                                                                                                                                                                                                                                                                                                                                                                           |
| `--format`       | `string` |         | Format output using a custom template:<br>'table':            Print output in table format with column headers (default)<br>'table TEMPLATE':   Print output in table format using the given Go template<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |
| `-n`, `--last`   | `int`    | `-1`    | Show n last created containers (includes all states)                                                                                                                                                                                                                                                                                                                                                                                 |
| `-l`, `--latest` |          |         | Show the latest created container (includes all states)                                                                                                                                                                                                                                                                                                                                                                              |
| `--no-trunc`     |          |         | Don't truncate output                                                                                                                                                                                                                                                                                                                                                                                                                |
| `-q`, `--quiet`  |          |         | Only display container IDs                                                                                                                                                                                                                                                                                                                                                                                                           |
| `-s`, `--size`   |          |         | Display total file sizes                                                                                                                                                                                                                                                                                                                                                                                                             |


<!---MARKER_GEN_END-->

