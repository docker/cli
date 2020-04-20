---
title: "volume inspect"
description: "The volume inspect command description and usage"
keywords: "volume, inspect"
---

# volume inspect

```markdown
Usage:  docker volume inspect [OPTIONS] VOLUME [VOLUME...]

Display detailed information on one or more volumes

Options:
  -f, --format string   Format the output using the given Go template
      --help            Print usage
```

## Description

Returns information about a volume. By default, this command renders all results
in a JSON array. You can specify an alternate format to execute a
given template for each result. Go's
[text/template](http://golang.org/pkg/text/template/) package describes all the
details of the format.

## Examples

```bash
$ docker volume create

8140a838303144125b4f54653b47ede0486282c623c3551fbc7f390cdc3e9cf5

$ docker volume inspect 85bffb0677236974f93955d8ecc4df55ef5070117b0e53333cc1b443777be24d

[
  {
    "CreatedAt": "2020-04-19T11:00:21Z",
    "Driver": "local",
    "Labels": {},
    "Mountpoint": "/var/lib/docker/volumes/8140a838303144125b4f54653b47ede0486282c623c3551fbc7f390cdc3e9cf5/_data",
    "Name": "8140a838303144125b4f54653b47ede0486282c623c3551fbc7f390cdc3e9cf5",
    "Options": {},
    "Scope": "local"
  }
]

$ docker volume inspect --format '{{ .Mountpoint }}' 8140a838303144125b4f54653b47ede0486282c623c3551fbc7f390cdc3e9cf5

/var/lib/docker/volumes/8140a838303144125b4f54653b47ede0486282c623c3551fbc7f390cdc3e9cf5/_data
```

## Related commands

* [volume create](volume_create.md)
* [volume ls](volume_ls.md)
* [volume rm](volume_rm.md)
* [volume prune](volume_prune.md)
* [Understand Data Volumes](https://docs.docker.com/storage/volumes/)
