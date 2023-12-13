# volume rm

<!---MARKER_GEN_START-->

Remove one or more volumes. You cannot remove a volume that is in use by a container.


### Aliases

`docker volume rm`, `docker volume remove`

### Options

| Name            | Type | Default | Description                              |
|:----------------|:-----|:--------|:-----------------------------------------|
| `-f`, `--force` |      |         | Force the removal of one or more volumes |


<!---MARKER_GEN_END-->

## Description

Remove one or more volumes. You can't remove a volume that's in use by a container.

## Examples

```console
$ docker volume rm hello

hello
```

## Related commands

* [volume create](volume_create.md)
* [volume inspect](volume_inspect.md)
* [volume ls](volume_ls.md)
* [volume prune](volume_prune.md)
* [Understand Data Volumes](https://docs.docker.com/storage/volumes/)
