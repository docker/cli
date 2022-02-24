---
title: "context export"
description: "The context export command description and usage"
keywords: "context, export"
---

# context export

```markdown
Usage:  docker context export [OPTIONS] CONTEXT [FILE|-]

Export a context to a tar archive FILE or a tar stream on STDOUT.
```

## Description

Exports a context to a file that can then be used with `docker context import`.

The default output filename is `<CONTEXT>.dockercontext`. To export to `STDOUT`, 
use `-` as filename, for example:

```console
$ docker context export my-context -
```
