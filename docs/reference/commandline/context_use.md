---
title: "context use"
description: "The context use command description and usage"
keywords: "context, use"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# context use

```markdown
Usage:  docker context use CONTEXT [OPTIONS]

Set the current docker context

Options:
      --skip-kubeconfig   Do not modify current kubeconfig file (set via
                          KUBECONFIG environment variable, or ~/.kube/config)
```

## Description
Set the default context to use, when `DOCKER_HOST`, `DOCKER_CONTEXT` environment variables and `--host`, `--context` global options are not set.

For contexts with a Kubernetes endpoint, this also modifes the current `kubeconfig` file to make `kubectl` and any other tool working with `kubeconfig` files target the same cluster.

