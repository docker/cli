# rmi

<!---MARKER_GEN_START-->
Remove one or more images

### Aliases

`docker image rm`, `docker image remove`, `docker rmi`

### Options

| Name                      | Type          | Default | Description                                                                                      |
|:--------------------------|:--------------|:--------|:-------------------------------------------------------------------------------------------------|
| `-f`, `--force`           | `bool`        |         | Force removal of the image                                                                       |
| `--no-prune`              | `bool`        |         | Do not delete untagged parents                                                                   |
| [`--platform`](#platform) | `stringSlice` |         | Remove only the given platform variant. Formatted as `os[/arch[/variant]]` (e.g., `linux/amd64`) |


<!---MARKER_GEN_END-->

## Description

Removes (and un-tags) one or more images from the host node. If an image has
multiple tags, using this command with the tag as a parameter only removes the
tag. If the tag is the only one for the image, both the image and the tag are
removed.

This does not remove images from a registry. You cannot remove an image of a
running container unless you use the `-f` option. To see all images on a host
use the [`docker image ls`](image_ls.md) command.

## Examples

You can remove an image using its short or long ID, its tag, or its digest. If
an image has one or more tags referencing it, you must remove all of them before
the image is removed. Digest references are removed automatically when an image
is removed by tag.

```console
$ docker images

REPOSITORY                TAG                 IMAGE ID            CREATED             SIZE
test1                     latest              fd484f19954f        23 seconds ago      7 B (virtual 4.964 MB)
test                      latest              fd484f19954f        23 seconds ago      7 B (virtual 4.964 MB)
test2                     latest              fd484f19954f        23 seconds ago      7 B (virtual 4.964 MB)

$ docker rmi fd484f19954f

Error: Conflict, cannot delete image fd484f19954f because it is tagged in multiple repositories, use -f to force
2013/12/11 05:47:16 Error: failed to remove one or more images

$ docker rmi test1:latest

Untagged: test1:latest

$ docker rmi test2:latest

Untagged: test2:latest


$ docker images

REPOSITORY                TAG                 IMAGE ID            CREATED             SIZE
test                      latest              fd484f19954f        23 seconds ago      7 B (virtual 4.964 MB)

$ docker rmi test:latest

Untagged: test:latest
Deleted: fd484f19954f4920da7ff372b5067f5b7ddb2fd3830cecd17b96ea9e286ba5b8
```

If you use the `-f` flag and specify the image's short or long ID, then this
command untags and removes all images that match the specified ID.

```console
$ docker images

REPOSITORY                TAG                 IMAGE ID            CREATED             SIZE
test1                     latest              fd484f19954f        23 seconds ago      7 B (virtual 4.964 MB)
test                      latest              fd484f19954f        23 seconds ago      7 B (virtual 4.964 MB)
test2                     latest              fd484f19954f        23 seconds ago      7 B (virtual 4.964 MB)

$ docker rmi -f fd484f19954f

Untagged: test1:latest
Untagged: test:latest
Untagged: test2:latest
Deleted: fd484f19954f4920da7ff372b5067f5b7ddb2fd3830cecd17b96ea9e286ba5b8
```

An image pulled by digest has no tag associated with it:

```console
$ docker images --digests

REPOSITORY                     TAG       DIGEST                                                                    IMAGE ID        CREATED         SIZE
localhost:5000/test/busybox    <none>    sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf   4986bf8c1536    9 weeks ago     2.43 MB
```

To remove an image using its digest:

```console
$ docker rmi localhost:5000/test/busybox@sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf
Untagged: localhost:5000/test/busybox@sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf
Deleted: 4986bf8c15363d1c5d15512d5266f8777bfba4974ac56e3270e7760f6f0a8125
Deleted: ea13149945cb6b1e746bf28032f02e9b5a793523481a0a18645fc77ad53c4ea2
Deleted: df7546f9f060a2268024c8a230d8639878585defcc1bc6f79d2728a13957871b
```

### <a name="platform"></a> Remove specific platforms (`--platform`)

The `--platform` option allows you to specify which platform variants of the
image to remove. By default, `docker image remove` removes all platform variants
that are present. Use the `--platform` option to specify which platform variant
of the image to remove.

Removing a specific platform removes the image from all images that reference
the same content, and requires the `--force` option to be used. Omitting the
`--force` option produces a warning, and the remove is canceled:

```console
$ docker image rm --platform=linux/amd64 alpine
Error response from daemon: Content will be removed from all images referencing this variant. Use —-force to force delete.
```

The platform option takes the `os[/arch[/variant]]` format; for example,
`linux/amd64` or `linux/arm64/v8`. Architecture and variant are optional,
and default to the daemon's native architecture if omitted.

You can pass multiple platforms either by passing the `--platform` flag
multiple times, or by passing a comma-separated list of platforms to remove.
The following uses of this option are equivalent;

```console
$ docker image rm --plaform linux/amd64 --platform linux/ppc64le myimage
$ docker image rm --plaform linux/amd64,linux/ppc64le myimage
```

The following example removes the `linux/amd64` and `linux/ppc64le` variants
of an `alpine` image that contains multiple platform variants in the image
cache:

```console
$ docker image ls --tree

IMAGE                   ID             DISK USAGE   CONTENT SIZE   EXTRA
alpine:latest           a8560b36e8b8       37.8MB         11.2MB    U
├─ linux/amd64          1c4eef651f65       12.1MB         3.64MB    U
├─ linux/arm/v6         903bfe2ae994           0B             0B
├─ linux/arm/v7         9c2d245b3c01           0B             0B
├─ linux/arm64/v8       757d680068d7       12.8MB         3.99MB
├─ linux/386            2436f2b3b7d2           0B             0B
├─ linux/ppc64le        9ed53fd3b831       12.8MB         3.58MB
├─ linux/riscv64        1de5eb4a9a67           0B             0B
└─ linux/s390x          fe0dcdd1f783           0B             0B
 
$ docker image --platform=linux/amd64,linux/ppc64le --force alpine
Deleted: sha256:1c4eef651f65e2f7daee7ee785882ac164b02b78fb74503052a26dc061c90474
Deleted: sha256:9ed53fd3b83120f78b33685d930ce9bf5aa481f6e2d165c42cbbddbeaa196f6f
```

After the command completes, the given variants of the `alpine` image are removed
from the image cache:

```console
$ docker image ls --tree

IMAGE                   ID             DISK USAGE   CONTENT SIZE   EXTRA
alpine:latest           a8560b36e8b8       12.8MB         3.99MB
├─ linux/amd64          1c4eef651f65           0B             0B
├─ linux/arm/v6         903bfe2ae994           0B             0B
├─ linux/arm/v7         9c2d245b3c01           0B             0B
├─ linux/arm64/v8       757d680068d7       12.8MB         3.99MB
├─ linux/386            2436f2b3b7d2           0B             0B
├─ linux/ppc64le        9ed53fd3b831           0B             0B
├─ linux/riscv64        1de5eb4a9a67           0B             0B
└─ linux/s390x          fe0dcdd1f783           0B             0B
```
