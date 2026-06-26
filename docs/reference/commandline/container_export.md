# export

<!---MARKER_GEN_START-->
Export a container's filesystem as a tar archive

### Aliases

`docker container export`, `docker export`

### Options

| Name             | Type     | Default | Description                        |
|:-----------------|:---------|:--------|:-----------------------------------|
| `-o`, `--output` | `string` |         | Write to a file, instead of STDOUT |


<!---MARKER_GEN_END-->

## Description

The `docker export` command doesn't export the contents of volumes associated
with the container. If a volume is mounted on top of an existing directory in
the container, `docker export` exports the contents of the underlying
directory, not the contents of the volume.

Refer to [Backup, restore, or migrate data volumes](https://docs.docker.com/engine/storage/volumes/#back-up-restore-or-migrate-data-volumes)
in the user guide for examples on exporting data in a volume.

## Examples

The following commands produce the same result.

```console
$ docker export red_panda > latest.tar
```

```console
$ docker export --output="latest.tar" red_panda
```
