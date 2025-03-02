# context create

<!---MARKER_GEN_START-->
Create a context

Docker endpoint config:

NAME                DESCRIPTION
from                Copy named context's Docker endpoint configuration
host                Docker endpoint on which to connect
ca                  Trust certs signed only by this CA
cert                Path to TLS certificate file
key                 Path to TLS key file
skip-tls-verify     Skip TLS certificate validation

Example:

$ docker context create my-context --description "some description" --docker "host=tcp://myserver:2376,ca=~/ca-file,cert=~/cert-file,key=~/key-file"


### Options

| Name                  | Type             | Default | Description                         |
|:----------------------|:-----------------|:--------|:------------------------------------|
| `--description`       | `string`         |         | Description of the context          |
| [`--docker`](#docker) | `stringToString` |         | set the docker endpoint             |
| [`--from`](#from)     | `string`         |         | create context from a named context |


<!---MARKER_GEN_END-->

## Description

Creates a new `context`. This lets you switch the daemon your `docker` CLI
connects to.

## Examples

### <a name="docker"></a> Create a context with a Docker endpoint (--docker)

Use the `--docker` flag to create a context with a custom endpoint. The
following example creates a context named `my-context` with a docker endpoint
of `/var/run/docker.sock`:

```console
$ docker context create \
    --docker host=unix:///var/run/docker.sock \
    my-context
```

### <a name="from"></a> Create a context based on an existing context (--from)

Use the `--from=<context-name>` option to create a new context from
an existing context. The example below creates a new context named `my-context`
from the existing context `existing-context`:

```console
$ docker context create --from existing-context my-context
```

If the `--from` option isn't set, the `context` is created from the current context:

```console
$ docker context create my-context
```

This can be used to create a context out of an existing `DOCKER_HOST` based script:

```console
$ source my-setup-script.sh
$ docker context create my-context
```

To source the `docker` endpoint configuration from an existing context
use the `--docker from=<context-name>` option. The example below creates a
new context named `my-context` using the docker endpoint configuration from
the existing context `existing-context`:

```console
$ docker context create \
    --docker from=existing-context \
    my-context
```

Docker endpoints configurations, as well as the description can be modified with
`docker context update`.

Refer to the [`docker context update` reference](context_update.md) for details.
