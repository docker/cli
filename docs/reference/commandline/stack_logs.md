# stack logs

<!---MARKER_GEN_START-->
Fetch aggregated logs of all services in the stack

### Options

| Name                 | Type     | Default | Description                                                                                     |
|:---------------------|:---------|:--------|:------------------------------------------------------------------------------------------------|
| `-f`, `--follow`     | `bool`   |         | Follow log output                                                                               |
| `--no-color`         | `bool`   |         | Produce monochrome output                                                                       |
| `--since`            | `string` |         | Show logs since timestamp (e.g. `2013-01-02T13:23:37Z`) or relative (e.g. `42m` for 42 minutes) |
| `-n`, `--tail`       | `string` | `all`   | Number of lines to show from the end of the logs (per service)                                  |
| `-t`, `--timestamps` | `bool`   |         | Show timestamps                                                                                 |


<!---MARKER_GEN_END-->

## Description

Fetches the logs of all services in the stack and aggregates them into a
single stream, prefixing every line with the name of the service it came
from. Each service gets its own color, similar to `docker compose logs`.
Use `--no-color` to produce monochrome output.

> [!NOTE]
> This command has to be run targeting a manager node.

## Examples

Follow the logs of all services in the stack `myapp`:

```console
$ docker stack logs --follow myapp
myapp_web      | 192.168.100.7 - - [03/Sep/2026:18:12:45 +0000] "GET / HTTP/1.1" 200 615
myapp_db       | 2026-09-03 18:12:46.017 UTC [1] LOG:  database system is ready to accept connections
myapp_worker   | processing job 42
```

Show the last 10 lines of each service, with timestamps:

```console
$ docker stack logs --tail 10 --timestamps myapp
```

## Related commands

* [stack deploy](stack_deploy.md)
* [stack ls](stack_ls.md)
* [stack ps](stack_ps.md)
* [stack rm](stack_rm.md)
* [stack services](stack_services.md)
