---
title: "volume rm"
description: "the volume rm command description and usage"
keywords: "volume, rm"
---

# volume rm

```markdown
Usage:  docker volume rm [OPTIONS] VOLUME [VOLUME...]

Remove one or more volumes

Aliases:
  rm, remove

Options:
  -f, --force  Force the removal of one or more volumes
      --help   Print usage
```

## Description

Remove one or more volumes. You cannot remove a volume that is in use by a container.

## Examples

```bash
$ docker volume rm hello

hello
```

## Related commands

* [volume create](volume_create.md)
* [volume inspect](volume_inspect.md)
* [volume ls](volume_ls.md)
* [volume prune](volume_prune.md)
* [Understand Data Volumes](https://docs.docker.com/storage/volumes/)
