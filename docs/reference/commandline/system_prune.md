---
title: "system prune"
description: "Remove unused data"
keywords: "system, prune, delete, remove"
---

# system prune

```markdown
Usage:	docker system prune [OPTIONS]

Remove unused data

Options:
  -a, --all             Remove all unused images not just dangling ones
      --filter filter   Provide filter values (e.g. 'label=<key>=<value>')
  -f, --force           Do not prompt for confirmation
      --help            Print usage
      --volumes         Prune volumes
```

## Description

Remove all unused containers, networks, images (both dangling and unreferenced),
and optionally, volumes.

## Examples

```bash
$ docker system prune

WARNING! This will remove:
        - all stopped containers
        - all networks not used by at least one container
        - all dangling images
        - all build cache
Are you sure you want to continue? [y/N] y

Deleted Containers:
f44f9b81948b3919590d5f79a680d8378f1139b41952e219830a33027c80c867
792776e68ac9d75bce4092bc1b5cc17b779bc926ab04f4185aec9bf1c0d4641f

Deleted Networks:
network1
network2

Deleted Images:
untagged: hello-world@sha256:f3b3b28a45160805bb16542c9531888519430e9e6d6ffc09d72261b0d26ff74f
deleted: sha256:1815c82652c03bfd8644afda26fb184f2ed891d921b20a0703b46768f9755c57
deleted: sha256:45761469c965421a92a69cc50e92c01e0cfa94fe026cdd1233445ea00e96289a

Total reclaimed space: 1.84kB
```

By default, volumes are not removed to prevent important data from being
deleted if there is currently no container using the volume. Use the `--volumes`
flag when running the command to prune volumes as well:

```bash
$ docker system prune -a --volumes

WARNING! This will remove:
        - all stopped containers
        - all networks not used by at least one container
        - all volumes not used by at least one container
        - all images without at least one container associated to them
        - all build cache
Are you sure you want to continue? [y/N] y

Deleted Containers:
0998aa37185a1a7036b0e12cf1ac1b6442dcfa30a5c9650a42ed5010046f195b
73958bfb884fa81fa4cc6baf61055667e940ea2357b4036acbbe25a60f442a4d

Deleted Networks:
my-network-a
my-network-b

Deleted Volumes:
named-vol

Deleted Images:
untagged: my-curl:latest
deleted: sha256:7d88582121f2a29031d92017754d62a0d1a215c97e8f0106c586546e7404447d
deleted: sha256:dd14a93d83593d4024152f85d7c63f76aaa4e73e228377ba1d130ef5149f4d8b
untagged: alpine:3.3
deleted: sha256:695f3d04125db3266d4ab7bbb3c6b23aa4293923e762aa2562c54f49a28f009f
untagged: alpine:latest
deleted: sha256:ee4603260daafe1a8c2f3b78fd760922918ab2441cbb2853ed5c439e59c52f96
deleted: sha256:9007f5987db353ec398a223bc5a135c5a9601798ba20a1abba537ea2f8ac765f
deleted: sha256:71fa90c8f04769c9721459d5aa0936db640b92c8c91c9b589b54abd412d120ab
deleted: sha256:bb1c3357b3c30ece26e6604aea7d2ec0ace4166ff34c3616701279c22444c0f3
untagged: my-jq:latest
deleted: sha256:6e66d724542af9bc4c4abf4a909791d7260b6d0110d8e220708b09e4ee1322e1
deleted: sha256:07b3fa89d4b17009eb3988dfc592c7d30ab3ba52d2007832dffcf6d40e3eda7f
deleted: sha256:3a88a5c81eb5c283e72db2dbc6d65cbfd8e80b6c89bb6e714cfaaa0eed99c548

Total reclaimed space: 13.5 MB
```

> **Note**
>
> The `--volumes` option was added in Docker 17.06.1. Older versions of Docker
> prune volumes by default, along with other Docker objects. On older versions,
> run `docker container prune`, `docker network prune`, and `docker image prune`
> separately to remove unused containers, networks, and images, without removing
> volumes.


### Filtering

The filtering flag (`--filter`) format is of "key=value". If there is more
than one filter, then pass multiple flags (e.g., `--filter "foo=bar" --filter "bif=baz"`)

The currently supported filters are:

* until (`<timestamp>`) - only remove containers, images, and networks created before given timestamp
* label (`label=<key>`, `label=<key>=<value>`, `label!=<key>`, or `label!=<key>=<value>`) - only remove containers, images, networks, and volumes with (or without, in case `label!=...` is used) the specified labels.

The `until` filter can be Unix timestamps, date formatted
timestamps, or Go duration strings (e.g. `10m`, `1h30m`) computed
relative to the daemon machine’s time. Supported formats for date
formatted time stamps include RFC3339Nano, RFC3339, `2006-01-02T15:04:05`,
`2006-01-02T15:04:05.999999999`, `2006-01-02Z07:00`, and `2006-01-02`. The local
timezone on the daemon will be used if you do not provide either a `Z` or a
`+-00:00` timezone offset at the end of the timestamp.  When providing Unix
timestamps enter seconds[.nanoseconds], where seconds is the number of seconds
that have elapsed since January 1, 1970 (midnight UTC/GMT), not counting leap
seconds (aka Unix epoch or Unix time), and the optional .nanoseconds field is a
fraction of a second no more than nine digits long.

```bash
$ docker system prune --filter 'until=10m'

WARNING! This will remove:
  - all stopped containers
  - all networks not used by at least one container
  - all dangling images
  - all dangling build cache

  Items to be pruned will be filtered with:
  - until=10m

Are you sure you want to continue? [y/N] y
Deleted Containers:
69c8e2d677f1f8858fbcd2bbe3be6a1723e0488c010c0491a2a1601b77832ec1
47232b10662f8d1d62dc056c6c9b937f975d57ddbec026aff1d5f2a6b1d8156e
67cbf4d5db4c492d7d9008302f12d67b081bba4c664867ee63d8f9252b248783
a5749c316e4912a59452a0f8547deca1923af9085f38f114f74c4b69d3962989
b10faab270db51790b8a58c9a6e164531633cbc5d2a7cb6ad7394a927607b72b
7cf63ab4b8b4386f1504b07933434fd13f1db80588a64c0659dad9aa5b1f9e52
99856eb7a03ce56d1dda9e174bde8460de335874b4a486ba6d3c38f1eac01c5a
bc8291585d8742b75f3e0e2294d5c9a21435c65b6fe5869c8c7f454c3c8aa29f
7aea29584e3cb33c3b05e0be4fe5396b9af16ab8106d625779d76624b3000e0d
fcfa5f8287d738d84494f830ad5367339e49fb4fec883c63f0c31f93675d7618
044c3caa2df6fb569db4c577efbf9d3b0f47254eae569f68fa2801aa73881bcd
723941fe18511357d141e946252f0412b6804038d58a7340eec0ed9f87f17181
efbf592b86e272133d1fc157ebcfc0d9d3ec1ae9abf0e69b29b6722219361667
ec91f0e99f3d7f6259338427b8b8c9e09b56ba3f891bb7381cff12e11a6ad7cf
f5a9e1b545a008881a019dbd3075268a74d3770ab56f7a1f3b0a46f5790741c7
e3d9ff619baf6a50c0e77c2af9538e0e2b092e86620f40c6c8e4629bc4b277b5

Deleted Networks:
network4
network6
network1
network3
network2
network5

Total reclaimed space: 0B
```

The `label` filter accepts two formats. One is the `label=...` (`label=<key>` or `label=<key>=<value>`),
which removes containers, images, networks, and volumes with the specified labels. The other
format is the `label!=...` (`label!=<key>` or `label!=<key>=<value>`), which removes
containers, images, networks, and volumes without the specified labels.

## Related commands

* [volume create](volume_create.md)
* [volume ls](volume_ls.md)
* [volume inspect](volume_inspect.md)
* [volume rm](volume_rm.md)
* [volume prune](volume_prune.md)
* [Understand Data Volumes](https://docs.docker.com/storage/volumes/)
* [system df](system_df.md)
* [container prune](container_prune.md)
* [image prune](image_prune.md)
* [network prune](network_prune.md)
* [system prune](system_prune.md)
