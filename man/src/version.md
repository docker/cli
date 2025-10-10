The version command prints the current version number for all independently
versioned Docker components.

# EXAMPLES

## Display Docker version information

The default output renders all version information divided into two sections;
the "Client" section contains information about the Docker CLI and client
components, and the "Server" section contains information about the Docker
Engine and components used by the Engine, such as the "Containerd" and "Runc"
OCI Runtimes.

The information shown may differ depending on how you installed Docker and
what components are in use. The following example shows the output on a macOS
machine running Docker Desktop:

    $ docker version
    Client: Docker Engine - Community
     Version:           28.5.1
     API version:       1.51
     Go version:        go1.24.8
     Git commit:        e180ab8
     Built:             Wed Oct  8 12:16:17 2025
     OS/Arch:           darwin/arm64
     Context:           desktop-linux
    
    Server: Docker Desktop 4.49.0 (12345)
     Engine:
      Version:          28.5.1
      API version:      1.51 (minimum version 1.24)
      Go version:       go1.24.8
      Git commit:       f8215cc
      Built:            Wed Oct  8 12:18:25 2025
      OS/Arch:          linux/arm64
      Experimental:     false
     containerd:
      Version:          1.7.27
      GitCommit:        05044ec0a9a75232cad458027ca83437aae3f4da
     runc:
      Version:          1.2.5
      GitCommit:        v1.2.5-0-g59923ef
     docker-init:
      Version:          0.19.0
      GitCommit:        de40ad0

Get server version:

    $ docker version --format '{{.Server.Version}}'
    28.5.1

Dump raw data:

To view all available fields, you can use the format `{{json .}}`.

    $ docker version --format '{{json .}}'
    {"Client":"Version":"28.5.1","ApiVersion":"1.51", ...}
