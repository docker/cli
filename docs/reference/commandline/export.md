---
title: "export"
description: "The export command description and usage"
keywords: "export, file, system, container"
---

# export

```markdown
Usage:  docker export [OPTIONS] CONTAINER

Export a container's filesystem as a tar archive

Options:
      --help            Print usage
  -o, --output string   Write to a file, instead of STDOUT
```

## Description

The `docker export` command does not export the contents of volumes associated
with the container. If a volume is mounted on top of an existing directory in
the container, `docker export` will export the contents of the *underlying*
directory, not the contents of the volume.

Refer to [Backup, restore, or migrate data volumes](https://docs.docker.com/storage/volumes/#backup-restore-or-migrate-data-volumes)
in the user guide for examples on exporting data in a volume.

## Examples

Each of these commands has the same result.

```console
$ docker export red_panda > latest.tar
```

```console
$ docker export --output="latest.tar" red_panda
```
