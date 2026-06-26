# volume prune

<!---MARKER_GEN_START-->
Remove unused local volumes

### Options

| Name                          | Type     | Default | Description                                        |
|:------------------------------|:---------|:--------|:---------------------------------------------------|
| [`-a`](#all), [`--all`](#all) | `bool`   |         | Remove all unused volumes, not just anonymous ones |
| [`--filter`](#filter)         | `filter` |         | Provide filter values (e.g. `label=<label>`)       |
| `-f`, `--force`               | `bool`   |         | Do not prompt for confirmation                     |


<!---MARKER_GEN_END-->

## Description

Remove all unused local volumes. Unused local volumes are those which are not
referenced by any containers. By default, it only removes anonymous volumes.

## Examples

```console
$ docker volume prune

WARNING! This will remove anonymous local volumes not used by at least one container.
Are you sure you want to continue? [y/N] y
Deleted Volumes:
07c7bdf3e34ab76d921894c2b834f073721fccfbbcba792aa7648e3a7a664c2e

Total reclaimed space: 36 B
```

### <a name="all"></a> Filtering (--all, -a)

Use the `--all` flag to prune both unused anonymous and named volumes.

### <a name="filter"></a> Filtering (--filter)

The filtering flag (`--filter`) format is of "key=value". If there is more
than one filter, then pass multiple flags (e.g., `--filter "foo=bar" --filter "bif=baz"`).

When multiple filters are provided, they are combined as follows:

- Multiple filters with **different keys** are combined using AND logic.
  A volume must satisfy all filter conditions to be pruned.
- Multiple filters with the **same key** are combined using OR logic.
  A volume is pruned if it matches any of the values for that key.

For example, `--filter "label=foo" --filter "label=bar"` prunes volumes that
have **either** the `foo` **or** `bar` label, while
`--filter "label=foo" --filter "label!=bar"` prunes volumes that have the
`foo` label **and** do not have the `bar` label.

The currently supported filters are:

* label (`label=<key>`, `label=<key>=<value>`, `label!=<key>`, or `label!=<key>=<value>`) - only remove volumes with (or without, in case `label!=...` is used) the specified labels.

The `label` filter accepts two formats. One is the `label=...` (`label=<key>` or `label=<key>=<value>`),
which removes volumes with the specified labels. The other
format is the `label!=...` (`label!=<key>` or `label!=<key>=<value>`), which removes
volumes without the specified labels.

## Related commands

* [volume create](volume_create.md)
* [volume ls](volume_ls.md)
* [volume inspect](volume_inspect.md)
* [volume rm](volume_rm.md)
* [Understand Data Volumes](https://docs.docker.com/storage/volumes/)
* [system df](system_df.md)
* [container prune](container_prune.md)
* [image prune](image_prune.md)
* [network prune](network_prune.md)
* [system prune](system_prune.md)
