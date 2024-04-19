# wait

<!---MARKER_GEN_START-->
Block until one or more containers stop, then print their exit codes

### Aliases

`docker container wait`, `docker wait`


<!---MARKER_GEN_END-->

> **Note**
>
> `docker wait` returns `0` when run against a container which had already
> exited before the `docker wait` command was run.

## Examples

Start a container in the background.

```console
$ docker run -dit --name=my_container ubuntu bash
```

Run `docker wait`, which should block until the container exits.

```console
$ docker wait my_container
```

In another terminal, stop the first container. The `docker wait` command above
returns the exit code.

```console
$ docker stop my_container
```

This is the same `docker wait` command from above, but it now exits, returning
`0`.

```console
$ docker wait my_container

0
```
