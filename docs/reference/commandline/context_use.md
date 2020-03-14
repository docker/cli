---
title: "context use"
description: "The context use command description and usage"
keywords: "context, use"
---

# context use

```markdown
Usage:  docker context use CONTEXT

Set the current docker context
```

## Description
Set the default context to use, when `DOCKER_HOST`, `DOCKER_CONTEXT` environment variables and `--host`, `--context` global options are not set.
To disable usage of contexts, you can use the special `default` context.