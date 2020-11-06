---
title: "context export"
description: "The context export command description and usage"
keywords: "context, export"
---

# context export

```markdown
Usage:  docker context export [OPTIONS] CONTEXT [FILE|-]

Export a context to a tar or kubeconfig file

Options:
      --kubeconfig   Export as a kubeconfig file
```

## Description

Exports a context in a file that can then be used with `docker context import`
(or with `kubectl` if `--kubeconfig` is set). Default output filename is
`<CONTEXT>.dockercontext`, or `<CONTEXT>.kubeconfig` if `--kubeconfig` is set.
To export to `STDOUT`, you can run `docker context export my-context -`.
