---
title: "unpause"
description: "The unpause command description and usage"
keywords: "cgroups, suspend, container"
---

# unpause

```markdown
Usage:  docker unpause CONTAINER [CONTAINER...]

Unpause all processes within one or more containers

Options:
      --help   Print usage
```

## Description

The `docker unpause` command un-suspends all processes in the specified containers.
On Linux, it does this using the freezer cgroup.

See the
[freezer cgroup documentation](https://www.kernel.org/doc/Documentation/cgroup-v1/freezer-subsystem.txt)
for further details.

## Examples

```console
$ docker unpause my_container
my_container
```

## Related commands

* [pause](pause.md)
