# import

<!---MARKER_GEN_START-->
Import the contents from a tarball to create a filesystem image

### Aliases

`docker image import`, `docker import`

### Options

| Name                                      | Type     | Default | Description                                       |
|:------------------------------------------|:---------|:--------|:--------------------------------------------------|
| [`-c`](#change), [`--change`](#change)    | `list`   |         | Apply Dockerfile instruction to the created image |
| [`-m`](#message), [`--message`](#message) | `string` |         | Set commit message for imported image             |
| [`--platform`](#platform)                 | `string` |         | Set platform if server is multi-platform capable  |


<!---MARKER_GEN_END-->

## Description

You can specify a `URL` or `-` (dash) to take data directly from `STDIN`. The
`URL` can point to an archive (.tar, .tar.gz, .tgz, .bzip, .tar.xz, or .txz)
containing a filesystem or to an individual file on the Docker host.  If you
specify an archive, Docker untars it in the container relative to the `/`
(root). If you specify an individual file, you must specify the full path within
the host. To import from a remote location, specify a `URI` that begins with the
`http://` or `https://` protocol.

## Examples

### Import from a remote location

This creates a new untagged image.

```console
$ docker import https://example.com/exampleimage.tgz
```

### Import from a local file

Import to docker via pipe and `STDIN`.

```console
$ cat exampleimage.tgz | docker import - exampleimagelocal:new
```

Import to docker from a local archive.

```console
$ docker import /path/to/exampleimage.tgz
```

### Import from a local directory

```console
$ sudo tar -c . | docker import - exampleimagedir
```

Note the `sudo` in this example – you must preserve
the ownership of the files (especially root ownership) during the
archiving with tar. If you are not root (or the sudo command) when you
tar, then the ownerships might not get preserved.

### <a name="change"></a> Import with new configurations (-c, --change)

The `--change` option applies `Dockerfile` instructions to the image that is
created. Not all `Dockerfile` instructions are supported; the list of instructions
is limited to metadata (configuration) changes. The following `Dockerfile`
instructions are supported:

- [`CMD`](https://docs.docker.com/reference/dockerfile/#cmd)
- [`ENTRYPOINT`](https://docs.docker.com/reference/dockerfile/#entrypoint)
- [`ENV`](https://docs.docker.com/reference/dockerfile/#env)
- [`EXPOSE`](https://docs.docker.com/reference/dockerfile/#expose)
- [`HEALTHCHECK`](https://docs.docker.com/reference/dockerfile/#healthcheck)
- [`LABEL`](https://docs.docker.com/reference/dockerfile/#label)
- [`ONBUILD`](https://docs.docker.com/reference/dockerfile/#onbuild)
- [`STOPSIGNAL`](https://docs.docker.com/reference/dockerfile/#stopsignal)
- [`USER`](https://docs.docker.com/reference/dockerfile/#user)
- [`VOLUME`](https://docs.docker.com/reference/dockerfile/#volume)
- [`WORKDIR`](https://docs.docker.com/reference/dockerfile/#workdir)

The following example imports an image from a TAR-file containing a root-filesystem,
and sets the `DEBUG` environment-variable in the resulting image:

```console
$ docker import --change "ENV DEBUG=true" ./rootfs.tgz exampleimagedir
```

The `--change` option can be set multiple times to apply multiple `Dockerfile`
instructions. The example below sets the `LABEL1` and `LABEL2` labels on
the imported image, in addition to the `DEBUG` environment variable from
the previous example:

```console
$ docker import \
    --change "ENV DEBUG=true" \
    --change "LABEL LABEL1=hello" \
    --change "LABEL LABEL2=world" \
    ./rootfs.tgz exampleimagedir
```

### <a name="message"></a> Import with a commit message (-m, --message)

The `--message`  (or `-m`) option allows you to set a custom comment in
the image's metadata. The following example imports an image from a local
archive and sets a custom message.

```console
$ docker import --message "New image imported from tarball" ./rootfs.tgz exampleimagelocal:new
sha256:25e54c0df7dc49da9093d50541e0ed4508a6b78705057f1a9bebf1d564e2cb00
```

After importing, the message is set in the "Comment" field of the image's
configuration, which is shown when viewing the image's history:

```console
$ docker image history exampleimagelocal:new

IMAGE          CREATED         CREATED BY   SIZE      COMMENT
25e54c0df7dc   2 minutes ago                53.6MB    New image imported from tarball
```

### When the daemon supports multiple operating systems

If the daemon supports multiple operating systems, and the image being imported
does not match the default operating system, it may be necessary to add
`--platform`. This would be necessary when importing a Linux image into a Windows
daemon.

```console
$ docker import --platform=linux .\linuximage.tar
```

### <a name="platform"></a> Set the platform for the imported image (--platform)

The `--platform` option allows you to specify the platform for the imported
image. By default, the daemon's native platform is used as platform, but
the `--platform` option allows you to override the default, for example, in
situations where the imported root filesystem is for a different architecture
or operating system.

The platform option takes the `os[/arch[/variant]]` format; for example,
`linux/amd64` or `linux/arm64/v8`. Architecture and variant are optional,
and default to the daemon's native architecture if omitted.

The following example imports an image from a root-filesystem in `rootfs.tgz`,
and sets the image's platform to `linux/amd64`;

```console
$ docker image import --platform=linux/amd64  ./rootfs.tgz imported:latest
sha256:44a8b44157dad5edcff85f0c93a3e455f3b20a046d025af4ec50ed990d7ebc09
```

After importing the image, the image's platform is set in the image's
configuration;

```console
$ docker image inspect --format '{{.Os}}/{{.Architecture}}' imported:latest
linux/amd64
```
