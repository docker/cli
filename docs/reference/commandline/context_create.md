---
title: "context create"
description: "The context create command description and usage"
keywords: "context, create"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# context create

```markdown
Usage:  docker context create [OPTIONS] CONTEXT

Create a context

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

$ docker context create my-context --description "some description" --docker "host=tcp://myserver:2376,ca=~/ca-file,cert=~/cert-file,key=~/key-file"

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

Creates a new `context`. This will allow you to quickly switch the cli configuration to connect to different clusters or single nodes.

To create a `context` out of an existing `DOCKER_HOST` based script, you can use the `from-current` config key:

```bash
$ source my-setup-script.sh
$ docker context create my-context --docker "from-current=true"
```

Similarly, to reference the currently active Kubernetes configuration, you can use `--kubernetes "from-current=true"`:

```bash
$ export KUBECONFIG=/path/to/my/kubeconfig
$ docker context create my-context --kubernetes "from-current=true" --docker "host=/var/run/docker.sock"
```

Docker and Kubernetes endpoints configurations, as well as default stack orchestrator and description can be modified with `docker context update`