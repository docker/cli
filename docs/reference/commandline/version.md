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
 Version:           28.5.1
 API version:       1.51
 Go version:        go1.24.8
 Git commit:        e180ab8
 Built:             Wed Oct  8 12:16:17 2025
 OS/Arch:           darwin/arm64
 Context:           remote-test-server

Server: Docker Desktop 4.19.0 (12345)
 Engine:
  Version:          27.5.1
  API version:      1.47 (minimum version 1.24)
  Go version:       go1.22.11
  Git commit:       4c9b3b0
  Built:            Wed Jan 22 13:41:24 2025
  OS/Arch:          linux/amd64
  Experimental:     true
 containerd:
  Version:          v1.7.25
  GitCommit:        bcc810d6b9066471b0b6fa75f557a15a1cbf31bb
 runc:
  Version:          1.2.4
  GitCommit:        v1.2.4-0-g6c52b3f
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
 Version:           28.5.1
 API version:       1.51
 Go version:        go1.24.8
 Git commit:        e180ab8
 Built:             Wed Oct  8 12:16:17 2025
 OS/Arch:           darwin/arm64
 Context:           remote-test-server

Server: Docker Engine - Community
 Engine:
  Version:          27.5.1
  API version:      1.47 (minimum version 1.24)
  Go version:       go1.22.11
  Git commit:       4c9b3b0
  Built:            Wed Jan 22 13:41:24 2025
  OS/Arch:          linux/amd64
  Experimental:     true
 containerd:
  Version:          v1.7.25
  GitCommit:        bcc810d6b9066471b0b6fa75f557a15a1cbf31bb
 runc:
  Version:          1.2.4
  GitCommit:        v1.2.4-0-g6c52b3f
 docker-init:
  Version:          0.19.0
  GitCommit:        de40ad0
```

### API version and version negotiation

The API version used by the client depends on the Docker Engine that the Docker
CLI is connecting with. When connecting with the Docker Engine, the Docker CLI
and Docker Engine perform API version negotiation, and select the highest API
version that is supported by both the Docker CLI and the Docker Engine.

For example, if the CLI is connecting with Docker Engine version 27.5, it downgrades
to API version 1.47 (refer to the [API version matrix](https://docs.docker.com/reference/api/engine/#api-version-matrix)
to learn about the supported API versions for Docker Engine):

```console
$ docker version --format '{{.Client.APIVersion}}'

1.47
```

Be aware that API version can also be overridden using the `DOCKER_API_VERSION`
environment variable, which can be useful for debugging purposes, and disables
API version negotiation. The following example illustrates an environment where
the `DOCKER_API_VERSION` environment variable is set. Unsetting the environment
variable removes the override, and re-enables API version negotiation:

```console
$ env | grep DOCKER_API_VERSION
DOCKER_API_VERSION=1.50

$ docker version --format '{{.Client.APIVersion}}'
1.50

$ unset DOCKER_API_VERSION
$ docker version --format '{{.Client.APIVersion}}'
1.51
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

28.5.1
```

### Get the client API version

The following example prints the API version that is used by the client:

```console
$ docker version --format '{{.Client.APIVersion}}'

1.51
```

The version shown is the API version that is negotiated between the client
and the Docker Engine. Refer to [API version and version negotiation](#api-version-and-version-negotiation)
above for more information.

### Dump raw JSON data

```console
$ docker version --format '{{json .}}'

{"Client":"Version":"28.5.1","ApiVersion":"1.51", ...}
```
