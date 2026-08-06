# docker logs

<!---MARKER_GEN_START-->
Fetch the logs of a container

### Aliases

`docker container logs`, `docker logs`

### Options

| Name                 | Type     | Default | Description                                                                                        |
|:---------------------|:---------|:--------|:---------------------------------------------------------------------------------------------------|
| `--details`          | `bool`   |         | Show extra details provided to logs                                                                |
| `-f`, `--follow`     | `bool`   |         | Follow log output                                                                                  |
| `--since`            | `string` |         | Show logs since timestamp (e.g. `2013-01-02T13:23:37Z`) or relative (e.g. `42m` for 42 minutes)    |
| `-n`, `--tail`       | `string` | `all`   | Number of lines to show from the end of the logs                                                   |
| `-t`, `--timestamps` | `bool`   |         | Show timestamps                                                                                    |
| `--until`            | `string` |         | Show logs before a timestamp (e.g. `2013-01-02T13:23:37Z`) or relative (e.g. `42m` for 42 minutes) |


In order to retrieve logs before a specific point in time in the future, run:

```bash
$ docker run --name test -d busybox sh -c "while true; do $(echo date); sleep 1; done"
$ date
Tue 14 Nov 2017 16:40:00 CET
$ docker logs -f --until=-2s test
Tue 14 Nov 2017 16:40:00 CET
Tue 14 Nov 2017 16:40:01 CET
Tue 14 Nov 2017 16:40:02 CET
```
<!---MARKER_GEN_END-->

