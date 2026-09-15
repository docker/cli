# docker start

<!---MARKER_GEN_START-->
Start one or more stopped containers

### Aliases

`docker container start`, `docker start`

### Options

| Name                      | Type       | Default | Description                                                                                                                                       |
|:--------------------------|:-----------|:--------|:--------------------------------------------------------------------------------------------------------------------------------------------------|
| `-a`, `--attach`          | `bool`     |         | Attach STDOUT/STDERR and forward signals                                                                                                          |
| `--checkpoint`            | `string`   |         | Restore from this checkpoint                                                                                                                      |
| `--checkpoint-dir`        | `string`   |         | Use a custom checkpoint storage directory                                                                                                         |
| `--detach-keys`           | `string`   |         | Override the key sequence for detaching a container                                                                                               |
| `-i`, `--interactive`     | `bool`     |         | Attach container's STDIN                                                                                                                          |
| `--start-healthy-timeout` | `duration` | `0s`    | Maximum time to wait for the containers to become healthy before returning, not allowed with --attach or --interactive (ms\|s\|m\|h) (default 0s) |


<!---MARKER_GEN_END-->

