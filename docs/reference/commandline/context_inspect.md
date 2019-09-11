---
title: "context inspect"
description: "The context inspect command description and usage"
keywords: "context, inspect"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# context inspect

```markdown
Usage:  docker context inspect [OPTIONS] [CONTEXT] [CONTEXT...]

Display detailed information on one or more contexts

Options:
  -f, --format string   Format the output using the given Go template
```

## Description

Inspects one or more contexts.

## Examples

### Inspect a context by name

```bash
$ docker context inspect "local+aks"

[
    {
        "Name": "local+aks",
        "Metadata": {
            "Description": "Local Docker Engine + Azure AKS endpoint",
            "StackOrchestrator": "kubernetes"
        },
        "Endpoints": {
            "docker": {
                "Host": "npipe:////./pipe/docker_engine",
                "SkipTLSVerify": false
            },
            "kubernetes": {
                "Host": "https://simon-aks-***.hcp.uksouth.azmk8s.io:443",
                "SkipTLSVerify": false,
                "DefaultNamespace": "default"
            }
        },
        "TLSMaterial": {
            "kubernetes": [
                "ca.pem",
                "cert.pem",
                "key.pem"
            ]
        },
        "Storage": {
            "MetadataPath": "C:\\Users\\simon\\.docker\\contexts\\meta\\cb6d08c0a1bfa5fe6f012e61a442788c00bed93f509141daff05f620fc54ddee",
            "TLSPath": "C:\\Users\\simon\\.docker\\contexts\\tls\\cb6d08c0a1bfa5fe6f012e61a442788c00bed93f509141daff05f620fc54ddee"
        }
    }
]
```