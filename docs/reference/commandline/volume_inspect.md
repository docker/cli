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
  -f, --format string   Format the output using the given Go template (default "json")
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
$ docker volume create myvolume

myvolume
```

Use the `docker volume inspect` comment to inspect the configuration of the volume:

```bash
$ docker volume inspect myvolume
```

The output is in JSON format, for example:

```json
[
  {
    "CreatedAt": "2020-04-19T11:00:21Z",
    "Driver": "local",
    "Labels": {},
    "Mountpoint": "/var/lib/docker/volumes/8140a838303144125b4f54653b47ede0486282c623c3551fbc7f390cdc3e9cf5/_data",
    "Name": "myvolume",
    "Options": {},
    "Scope": "local"
  }
]
```
Use the `--format` flag to format the output using a Go template, for example,
to print the `Mountpoint` property:

```bash
$ docker volume inspect --format '{{ .Mountpoint }}' myvolume

/var/lib/docker/volumes/myvolume/_data
```

## Related commands

* [volume create](volume_create.md)
* [volume ls](volume_ls.md)
* [volume rm](volume_rm.md)
* [volume prune](volume_prune.md)
* [Understand Data Volumes](https://docs.docker.com/storage/volumes/)
