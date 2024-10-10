# load

<!---MARKER_GEN_START-->
Load an image from a tar archive or STDIN

### Aliases

`docker image load`, `docker load`

### Options

| Name                                | Type     | Default | Description                                                                                    |
|:------------------------------------|:---------|:--------|:-----------------------------------------------------------------------------------------------|
| [`-i`](#input), [`--input`](#input) | `string` |         | Read from tar archive file, instead of STDIN                                                   |
| [`--platform`](#platform)           | `string` |         | Load only the given platform variant. Formatted as `os[/arch[/variant]]` (e.g., `linux/amd64`) |
| `-q`, `--quiet`                     | `bool`   |         | Suppress the load output                                                                       |


<!---MARKER_GEN_END-->

## Description

Load an image or repository from a tar archive (even if compressed with gzip,
bzip2, xz or zstd) from a file or STDIN. It restores both images and tags.

## Examples

```console
$ docker image ls

REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
```

### Load images from STDIN

```console
$ docker load < busybox.tar.gz

Loaded image: busybox:latest
$ docker images
REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
busybox             latest              769b9341d937        7 weeks ago         2.489 MB
```

### <a name="input"></a> Load images from a file (--input)

```console
$ docker load --input fedora.tar

Loaded image: fedora:rawhide
Loaded image: fedora:20

$ docker images

REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
busybox             latest              769b9341d937        7 weeks ago         2.489 MB
fedora              rawhide             0d20aec6529d        7 weeks ago         387 MB
fedora              20                  58394af37342        7 weeks ago         385.5 MB
fedora              heisenbug           58394af37342        7 weeks ago         385.5 MB
fedora              latest              58394af37342        7 weeks ago         385.5 MB
```


### <a name="platform"></a> Load a specific platform (--platform)

The `--platform` option allows you to specify which platform variant of the
image to load. By default, `docker load` loads all platform variants that
are present in the archive. Use the `--platform` option to specify which
platform variant of the image to load. An error is produced if the given
platform is not present in the archive.

The platform option takes the `os[/arch[/variant]]` format; for example,
`linux/amd64` or `linux/arm64/v8`. Architecture and variant are optional,
and default to the daemon's native architecture if omitted.

The following example loads the `linux/amd64` variant of an `alpine` image
from an archive that contains multiple platform variants.

```console
$ docker image load -i image.tar --platform=linux/amd64
Loaded image: alpine:latest
```

The following example attempts to load a `linux/ppc64le` image from an
archive, but the given platform is not present in the archive;

```console
$ docker image load -i image.tar --platform=linux/ppc64le
requested platform (linux/ppc64le) not found: image might be filtered out
```
