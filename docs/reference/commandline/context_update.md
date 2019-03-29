---
title: "context update"
description: "The context update command description and usage"
keywords: "context, update"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# context update

```markdown
Usage:  docker context update [OPTIONS] CONTEXT

Update a context

Docker endpoint config:

NAME                DESCRIPTION
from-current        Copy current Docker endpoint configuration
host                Docker endpoint on which to connect
ca                  Trust certs signed only by this CA
cert                Path to TLS certificate file
key                 Path to TLS key file
skip-tls-verify     Skip TLS certificate validation

Kubernetes endpoint config:

NAME                 DESCRIPTION
from-current         Copy current Kubernetes endpoint configuration
config-file          Path to a Kubernetes config file
context-override     Overrides the context set in the kubernetes config file
namespace-override   Overrides the namespace set in the kubernetes config file

Example:

$ docker context update my-context --description "some description" --docker "host=tcp://myserver:2376,ca=~/ca-file,cert=~/cert-file,key=~/key-file"

Options:
      --default-stack-orchestrator string   Default orchestrator for
                                            stack operations to use with
                                            this context
                                            (swarm|kubernetes|all)
      --description string                  Description of the context
      --docker stringToString               set the docker endpoint
                                            (default [])
      --kubernetes stringToString           set the kubernetes endpoint
                                            (default [])
```

## Description

Updates an existing `context`.
See [context create](context_create.md)