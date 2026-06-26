# history

<!---MARKER_GEN_START-->
Show the history of an image

### Aliases

`docker image history`, `docker history`

### Options

| Name                      | Type     | Default | Description                                                                                                                                                                                                                                                                                                                                                                                                                          |
|:--------------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`--format`](#format)     | `string` |         | Format output using a custom template:<br>'table':            Print output in table format with column headers (default)<br>'table TEMPLATE':   Print output in table format using the given Go template<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |
| `-H`, `--human`           | `bool`   | `true`  | Print sizes and dates in human readable format                                                                                                                                                                                                                                                                                                                                                                                       |
| `--no-trunc`              | `bool`   |         | Don't truncate output                                                                                                                                                                                                                                                                                                                                                                                                                |
| [`--platform`](#platform) | `string` |         | Show history for the given platform. Formatted as `os[/arch[/variant]]` (e.g., `linux/amd64`)                                                                                                                                                                                                                                                                                                                                        |
| `-q`, `--quiet`           | `bool`   |         | Only show image IDs                                                                                                                                                                                                                                                                                                                                                                                                                  |


<!---MARKER_GEN_END-->

## Examples

To see how the `docker:latest` image was built:

```console
$ docker history docker

IMAGE               CREATED             CREATED BY                                      SIZE                COMMENT
3e23a5875458        8 days ago          /bin/sh -c #(nop) ENV LC_ALL=C.UTF-8            0 B
8578938dd170        8 days ago          /bin/sh -c dpkg-reconfigure locales &&    loc   1.245 MB
be51b77efb42        8 days ago          /bin/sh -c apt-get update && apt-get install    338.3 MB
4b137612be55        6 weeks ago         /bin/sh -c #(nop) ADD jessie.tar.xz in /        121 MB
750d58736b4b        6 weeks ago         /bin/sh -c #(nop) MAINTAINER Tianon Gravi <ad   0 B
511136ea3c5a        9 months ago                                                        0 B                 Imported from -
```

To see how the `docker:apache` image was added to a container's base image:

```console
$ docker history docker:scm
IMAGE               CREATED             CREATED BY                                      SIZE                COMMENT
2ac9d1098bf1        3 months ago        /bin/bash                                       241.4 MB            Added Apache to Fedora base image
88b42ffd1f7c        5 months ago        /bin/sh -c #(nop) ADD file:1fd8d7f9f6557cafc7   373.7 MB
c69cab00d6ef        5 months ago        /bin/sh -c #(nop) MAINTAINER Lokesh Mandvekar   0 B
511136ea3c5a        19 months ago                                                       0 B                 Imported from -
```

### <a name="format"></a> Format the output (--format)

The formatting option (`--format`) will pretty-prints history output
using a Go template.

Valid placeholders for the Go template are listed below:

| Placeholder     | Description                                                                                               |
|-----------------|-----------------------------------------------------------------------------------------------------------|
| `.ID`           | Image ID                                                                                                  |
| `.CreatedSince` | Elapsed time since the image was created if `--human=true`, otherwise timestamp of when image was created |
| `.CreatedAt`    | Timestamp of when image was created                                                                       |
| `.CreatedBy`    | Command that was used to create the image                                                                 |
| `.Size`         | Image disk size                                                                                           |
| `.Comment`      | Comment for image                                                                                         |

When using the `--format` option, the `history` command either
outputs the data exactly as the template declares or, when using the
`table` directive, includes column headers as well.

The following example uses a template without headers and outputs the
`ID` and `CreatedSince` entries separated by a colon (`:`) for the `busybox`
image:

```console
$ docker history --format "{{.ID}}: {{.CreatedSince}}" busybox

f6e427c148a7: 4 weeks ago
<missing>: 4 weeks ago
```

### <a name="platform"></a> Show history for a specific platform (--platform)

The `--platform` option allows you to specify which platform variant to show
history for if multiple platforms are present. By default, `docker history`
shows the history for the daemon's native platform or if not present, the
first available platform.

If the local image store has multiple platform variants of an image, the
`--platform` option selects which variant to show the history for. An error
is produced if the given platform is not present in the local image cache.

The platform option takes the `os[/arch[/variant]]` format; for example,
`linux/amd64` or `linux/arm64/v8`. Architecture and variant are optional,
and if omitted falls back to the daemon's defaults.


The following example pulls the RISC-V variant of the `alpine:latest` image
and shows its history.


```console
$ docker image pull --quiet --platform=linux/riscv64 alpine
docker.io/library/alpine:latest

$ docker image history --platform=linux/s390x alpine
IMAGE          CREATED       CREATED BY                                      SIZE      COMMENT
beefdbd8a1da   3 weeks ago   /bin/sh -c #(nop)  CMD ["/bin/sh"]              0B
<missing>      3 weeks ago   /bin/sh -c #(nop) ADD file:ba2637314e600db5a…   8.46MB
```

The following example attempts to show the history for a platform variant of
`alpine:latest` that doesn't exist in the local image store, resulting in
an error.

```console
$ docker image ls --tree
IMAGE                   ID             DISK USAGE   CONTENT SIZE   IN USE
alpine:latest           beefdbd8a1da       10.6MB         3.37MB
├─ linux/riscv64        80cde017a105       10.6MB         3.37MB
├─ linux/amd64          33735bd63cf8           0B             0B
├─ linux/arm/v6         50f635c8b04d           0B             0B
├─ linux/arm/v7         f2f82d424957           0B             0B
├─ linux/arm64/v8       9cee2b382fe2           0B             0B
├─ linux/386            b3e87f642f5c           0B             0B
├─ linux/ppc64le        c7a6800e3dc5           0B             0B
└─ linux/s390x          2b5b26e09ca2           0B             0B

$ docker image history --platform=linux/s390x alpine
Error response from daemon: image with reference alpine:latest was found but does not match the specified platform: wanted linux/s390x
```
