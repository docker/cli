---
title: "context ls"
description: "The context ls command description and usage"
keywords: "context, ls"
---

# context ls

```markdown
Usage:  docker context ls [OPTIONS]

List contexts

Aliases:
  ls, list

Options:
      --format string   Pretty-print contexts using a Go template
                        (default "table")
  -q, --quiet           Only show context names
```

## Examples

Use `docker context ls` to print all contexts. The currently active context is
indicated with an `*`:

```bash
NAME                DESCRIPTION                               DOCKER ENDPOINT                      KUBERNETES ENDPOINT   ORCHESTRATOR
default *           Current DOCKER_HOST based configuration   unix:///var/run/docker.sock                                swarm
production                                                    tcp:///prod.corp.example.com:2376
staging                                                       tcp:///stage.corp.example.com:2376
```
