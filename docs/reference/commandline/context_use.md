# context use

Set the default context for the Docker CLI.

## Usage

    docker context use CONTEXT

## Description

`docker context use` sets the **default context** for the Docker CLI.

This command updates the Docker CLI configuration (the `currentContext` field in the client configuration file, typically `~/.docker/config.json`). Because it's a configuration change, it is **sticky** and affects **all terminal sessions** that use the same Docker CLI config directory.

To change the context only for a single command, or only for your current shell session, use `--context` or the `DOCKER_CONTEXT` environment variable instead.

## Examples

### Set the default (sticky) context

This updates the CLI configuration and applies to new terminal sessions:

    $ docker context use chocolate
    chocolate

    $ docker context show
    chocolate

### Use a context for a single command

Use the global `--context` flag to avoid changing the default:

    $ docker --context chocolate ps

### Use a context for the current shell session

Set `DOCKER_CONTEXT` to override the configured default in the current shell:

    $ export DOCKER_CONTEXT=chocolate
    $ docker context show
    chocolate

To stop overriding:

    $ unset DOCKER_CONTEXT

### Switch back to the default context

    $ docker context use default
    default
