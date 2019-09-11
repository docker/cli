---
title: "context export"
description: "The context export command description and usage"
keywords: "context, export"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# context export

```markdown
Usage:  docker context export [OPTIONS] CONTEXT [FILE|-]

Export a context to a tar or kubeconfig file

Options:
      --kubeconfig   Export as a kubeconfig file
```

## Description

Exports a context in a file that can then be used with `docker context import` (or with `kubectl` if `--kubeconfig` is set).
Default output filename is `<CONTEXT>.dockercontext`, or `<CONTEXT>.kubeconfig` if `--kubeconfig` is set.
To export to `STDOUT`, you can run `docker context export my-context -`.
