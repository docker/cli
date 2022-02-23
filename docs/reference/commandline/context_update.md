---
title: "context update"
description: "The context update command description and usage"
keywords: "context, update"
---

# context update

```markdown
Usage:  docker context update [OPTIONS] CONTEXT

Update a context

Docker endpoint config:

NAME                DESCRIPTION
from                Copy Docker endpoint configuration from an existing context
host                Docker endpoint on which to connect
ca                  Trust certs signed only by this CA
cert                Path to TLS certificate file
key                 Path to TLS key file
skip-tls-verify     Skip TLS certificate validation

Example:

$ docker context update my-context --description "some description" --docker "host=tcp://myserver:2376,ca=~/ca-file,cert=~/cert-file,key=~/key-file"

Options:
      --description string                  Description of the context
      --docker stringToString               set the docker endpoint
                                            (default [])
```

## Description

Updates an existing `context`.
See [context create](context_create.md).

## Examples

### Update an existing context

```console
$ docker context update \
    --description "some description" \
    --docker "host=tcp://myserver:2376,ca=~/ca-file,cert=~/cert-file,key=~/key-file" \
    my-context
```
