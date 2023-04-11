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

Get server version:

    $ docker version --format '{{.Server.Version}}'
    23.0.3

Dump raw data:

To view all available fields, you can use the format `{{json .}}`.

    $ docker version --format '{{json .}}'
    {"Client":"Version":"23.0.3","ApiVersion":"1.42", ...}
