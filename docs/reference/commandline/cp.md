# docker cp

<!---MARKER_GEN_START-->
Copy files/folders between a container and the local filesystem

Use '-' as the source to read a tar archive from stdin
and extract it to a directory destination in a container.
Use '-' as the destination to stream a tar archive of a
container source to stdout.

### Aliases

`docker container cp`, `docker cp`

### Options

| Name                  | Type | Default | Description                                                                                                  |
|:----------------------|:-----|:--------|:-------------------------------------------------------------------------------------------------------------|
| `-a`, `--archive`     |      |         | Archive mode (copy all uid/gid information)                                                                  |
| `-L`, `--follow-link` |      |         | Always follow symbol link in SRC_PATH                                                                        |
| `-q`, `--quiet`       |      |         | Suppress progress output during copy. Progress output is automatically suppressed if no terminal is attached |


<!---MARKER_GEN_END-->

