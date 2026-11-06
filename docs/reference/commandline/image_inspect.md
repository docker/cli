# image inspect

<!---MARKER_GEN_START-->
Display detailed information on one or more images

### Options

| Name             | Type     | Default | Description                                                                                                                                                                                                                                                        |
|:-----------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-f`, `--format` | `string` |         | Format output using a custom template:<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |
| `--platform`     | `string` |         | Inspect a specific platform of the multi-platform image.<br>If the image or the server is not multi-platform capable, the command will error out if the platform does not match.<br>'os[/arch[/variant]]': Explicit platform (eg. linux/amd64)                     |


<!---MARKER_GEN_END-->

## Description

Print the image record Docker has locally: IDs and tags, platform, size, and
the image `Config` (default command, env, labels, and so on).

Use `--format` to pick fields instead of dumping the whole JSON.

## Examples

```console
$ docker image inspect alpine
```

Common fields:

- `Id`, `RepoTags`, `RepoDigests`
- `Architecture`, `Os`, `Size`
- `Config`: image config (`Env`, `Cmd`, `Entrypoint`, `WorkingDir`, `Labels`, `ExposedPorts`)

```console
$ docker image inspect --format '{{.Os}}/{{.Architecture}}' alpine
linux/amd64

$ docker image inspect --format '{{json .Config.Env}}' alpine
["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"]
```

`Config` follows the
[OCI image configuration](https://github.com/opencontainers/image-spec/blob/main/config.md).
