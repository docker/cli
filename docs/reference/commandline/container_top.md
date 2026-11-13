# top

<!---MARKER_GEN_START-->
Display the running processes of a container

### Aliases

`docker container top`, `docker top`


<!---MARKER_GEN_END-->

## Description

The `docker top` command shows the processes running inside a container.

On Linux, extra arguments after the container name are passed through to
`ps`. On Windows, those extra arguments aren't supported.

## Examples

### Show processes in a container

```console
$ docker top my-container
UID                 PID                 PPID                C                   STIME               TTY                 TIME                CMD
root                1853                1838                0                   12:01               ?                   00:00:00            nginx: master process nginx -g daemon off;
```

### Pass options through to ps (Linux)

```console
$ docker top my-container -o pid,comm
PID                 COMMAND
1853                nginx
```
