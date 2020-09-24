---
title: "stack config"
description: "The stack config command description and usage"
keywords: "stack, config"
---

# stack config

```markdown
Usage:	docker stack config [OPTIONS]

Outputs the final config file, after doing merges and interpolations

Aliases:
  config, cfg

Options:
  -c, --compose-file strings   Path to a Compose file, or "-" to read from stdin
      --orchestrator string    Orchestrator to use (swarm|kubernetes|all)
      --skip-interpolation     Skip interpolation and output only merged config
```

## Description

Outputs the final Compose file, after doing the merges and interpolations of the input Compose files.

## Examples

The following command outputs the result of the merge and interpolation of two Compose files.

```bash
$ docker stack config --compose-file docker-compose.yml --compose-file docker-compose.prod.yml
```

The Compose file can also be provided as standard input with `--compose-file -`:

```bash
$ cat docker-compose.yml | docker stack config --compose-file -
```

### Skipping interpolation

In some cases, it might be useful to skip interpolation of environment variables.
For example, when you want to pipe the output of this command back to `stack deploy`.

If you have a regex for a redirect route in an environment variable for your webserver you would use two `$` signs to prevent `stack deploy` from interpolating `${1}`.

```bash
  service: webserver
  environment:
    REDIRECT_REGEX=http://host/redirect/$${1} 
```

With interpolation, the `stack config` command will replace the environment variable in the Compose file 
with `REDIRECT_REGEX=http://host/redirect/${1}`, but then when piping it back to the `stack deploy` 
command it will be interpolated again and result in undefined behavior. 
That is why, when piping the output back to `stack deploy` one should always prefer the `--skip-interpolation` option.

```
$ docker stack config --compose-file web.yml --compose-file web.prod.yml --skip-interpolation | docker stack deploy --compose-file -
```

## Related commands

* [stack deploy](stack_deploy.md)
* [stack ps](stack_ps.md)
* [stack rm](stack_rm.md)
* [stack services](stack_services.md)
