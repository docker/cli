# version

<!---MARKER_GEN_START-->
Show the Docker version information

### Options

| Name                                   | Type     | Default | Description                                                                                                                                                                                                                                                        |
|:---------------------------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`-f`](#format), [`--format`](#format) | `string` |         | Format output using a custom template:<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |


<!---MARKER_GEN_END-->

## Description

The version command prints the current version number for all independently
versioned Docker components. Use the [`--format`](#format) option to customize
the output.

The version command (`docker version`) outputs the version numbers of Docker
components, while the `--version` flag (`docker --version`) outputs the version
number of the Docker CLI you are using.

### Default output

The default output renders all version information divided into two sections;
the `Client` section contains information about the Docker CLI and client
components, and the `Server` section contains information about the Docker
Engine and components used by the Docker Engine, such as the containerd and runc
OCI Runtimes.

The information shown may differ depending on how you installed Docker and
what components are in use. The following example shows the output on a macOS
machine running Docker Desktop:

```console
$ docker version

Client: Docker Engine - Community
 Version:           23.0.3
 API version:       1.42
 Go version:        go1.19.7
 Git commit:        3e7cbfd
 Built:             Tue Apr  4 22:05:41 2023
 OS/Arch:           darwin/amd64
 Context:           default

Server: Docker Desktop 4.19.0 (12345)
 Engine:
  Version:          23.0.3
  API version:      1.42 (minimum version 1.12)
  Go version:       go1.19.7
  Git commit:       59118bf
  Built:            Tue Apr  4 22:05:41 2023
  OS/Arch:          linux/amd64
  Experimental:     false
 containerd:
  Version:          1.6.20
  GitCommit:        2806fc1057397dbaeefbea0e4e17bddfbd388f38
 runc:
  Version:          1.1.5
  GitCommit:        v1.1.5-0-gf19387a
 docker-init:
  Version:          0.19.0
  GitCommit:        de40ad0
```

### Client and server versions

Docker uses a client/server architecture, which allows you to use the Docker CLI
on your local machine to control a Docker Engine running on a remote machine,
which can be (for example) a machine running in the cloud or inside a virtual machine.

The following example switches the Docker CLI to use a [context](context.md)
named `remote-test-server`, which runs an older version of the Docker Engine
on a Linux server:

```console
$ docker context use remote-test-server
remote-test-server

$ docker version

Client: Docker Engine - Community
 Version:           23.0.3
 API version:       1.40 (downgraded from 1.42)
 Go version:        go1.19.7
 Git commit:        3e7cbfd
 Built:             Tue Apr  4 22:05:41 2023
 OS/Arch:           darwin/amd64
 Context:           remote-test-server

Server: Docker Engine - Community
 Engine:
  Version:          19.03.8
  API version:      1.40 (minimum version 1.12)
  Go version:       go1.12.17
  Git commit:       afacb8b
  Built:            Wed Mar 11 01:29:16 2020
  OS/Arch:          linux/amd64
 containerd:
  Version:          v1.2.13
  GitCommit:        7ad184331fa3e55e52b890ea95e65ba581ae3429
 runc:
  Version:          1.0.0-rc10
  GitCommit:        dc9208a3303feef5b3839f4323d9beb36df0a9dd
 docker-init:
  Version:          0.18.0
  GitCommit:        fec3683
```

### API version and version negotiation

The API version used by the client depends on the Docker Engine that the Docker
CLI is connecting with. When connecting with the Docker Engine, the Docker CLI
and Docker Engine perform API version negotiation, and select the highest API
version that is supported by both the Docker CLI and the Docker Engine.

For example, if the CLI is connecting with Docker Engine version 19.03, it downgrades
to API version 1.40 (refer to the [API version matrix](https://docs.docker.com/reference/api/engine/#api-version-matrix)
to learn about the supported API versions for Docker Engine):

```console
$ docker version --format '{{.Client.APIVersion}}'

1.40
```

Be aware that API version can also be overridden using the `DOCKER_API_VERSION`
environment variable, which can be useful for debugging purposes, and disables
API version negotiation. The following example illustrates an environment where
the `DOCKER_API_VERSION` environment variable is set. Unsetting the environment
variable removes the override, and re-enables API version negotiation:

```console
$ env | grep DOCKER_API_VERSION
DOCKER_API_VERSION=1.39

$ docker version --format '{{.Client.APIVersion}}'
1.39

$ unset DOCKER_API_VERSION
$ docker version --format '{{.Client.APIVersion}}'
1.42
```

## Examples

### <a name="format"></a> Format the output (--format)

The formatting option (`--format`) pretty-prints the output using a Go template,
which allows you to customize the output format, or to obtain specific information
from the output. Refer to the [format command and log output](https://docs.docker.com/config/formatting/)
page for details of the format.

### Get the server version

```console
$ docker version --format '{{.Server.Version}}'

23.0.3
```

### Get the client API version

The following example prints the API version that is used by the client:

```console
$ docker version --format '{{.Client.APIVersion}}'

1.42
```

The version shown is the API version that is negotiated between the client
and the Docker Engine. Refer to [API version and version negotiation](#api-version-and-version-negotiation)
above for more information.

### Dump raw JSON data

```console
$ docker version --format '{{json .}}'

{"Client":"Version":"23.0.3","ApiVersion":"1.42", ...}
```
