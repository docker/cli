# context use

<!---MARKER_GEN_START-->
Set the default docker context


<!---MARKER_GEN_END-->

## Description

The `docker context use` command sets the default context for the Docker CLI.

The `docker context use` command sets the Docker CLI’s default context by updating
your CLI config (`~/.docker/config.json`). This change is persistent, affecting
all shells and sessions that share that config, not just the current terminal.

For one-off commands or per-shell usage, use `--context` or the `DOCKER_CONTEXT`
environment variable instead.

## Examples

### Set the default (sticky) context

This updates the CLI configuration and applies to new terminal sessions:

```bash
$ docker context use my-context
my-context

$ docker context show
my-context
```

### Use a context for a single command

Use the global `--context` flag to avoid changing the default:

```bash
$ docker --context my-context ps
```

### Use a context for the current shell session

Set `DOCKER_CONTEXT` to override the configured default in the current shell:

```bash
$ export DOCKER_CONTEXT=my-context
$ docker context show
my-context
```

To stop overriding:

```bash
$ unset DOCKER_CONTEXT
```

### Switch back to the default context

```bash
$ docker context use default
default
```
