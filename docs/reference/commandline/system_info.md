# info

<!---MARKER_GEN_START-->
Display system-wide information

### Aliases

`docker system info`, `docker info`

### Options

| Name                                   | Type     | Default | Description                                                                                                                                                                                                                                                        |
|:---------------------------------------|:---------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`-f`](#format), [`--format`](#format) | `string` |         | Format output using a custom template:<br>'json':             Print in JSON format<br>'TEMPLATE':         Print output using the given Go template.<br>Refer to https://docs.docker.com/go/formatting/ for more information about formatting output with templates |


<!---MARKER_GEN_END-->

## Description

This command displays system wide information regarding the Docker installation.
Information displayed includes the kernel version, number of containers and images.
The number of images shown is the number of unique images. The same image tagged
under different names is counted only once.

If a format is specified, the given template will be executed instead of the
default format. Go's [text/template](https://pkg.go.dev/text/template) package
describes all the details of the format.

Depending on the storage driver in use, additional information can be shown, such
as pool name, data file, metadata file, data space used, total data space, metadata
space used, and total metadata space.

The data file is where the images are stored and the metadata file is where the
meta data regarding those images are stored. When run for the first time Docker
allocates a certain amount of data space and meta data space from the space
available on the volume where `/var/lib/docker` is mounted.

## Examples

### Show output

The example below shows the output for a daemon running on Ubuntu Linux,
using the `overlay2` storage driver. As can be seen in the output, additional
information about the `overlay2` storage driver is shown:

```console
$ docker info

Client:
 Version:    25.0.0
 Context:    default
 Debug Mode: false
 Plugins:
  buildx: Docker Buildx (Docker Inc.)
    Version:  v0.12.1
    Path:     /usr/local/libexec/docker/cli-plugins/docker-buildx
  compose: Docker Compose (Docker Inc.)
    Version:  v2.24.1
    Path:     /usr/local/libexec/docker/cli-plugins/docker-compose

Server:
 Containers: 14
  Running: 3
  Paused: 1
  Stopped: 10
 Images: 52
 Server Version: 25.0.0
 Storage Driver: overlayfs
  driver-type: io.containerd.snapshotter.v1
 Logging Driver: json-file
 Cgroup Driver: cgroupfs
 Cgroup Version: 2
 Plugins:
  Volume: local
  Network: bridge host ipvlan macvlan null overlay
  Log: awslogs fluentd gcplogs gelf journald json-file local splunk syslog
 CDI spec directories:
  /etc/cdi
  /var/run/cdi
 Swarm: inactive
 Runtimes: runc io.containerd.runc.v2
 Default Runtime: runc
 Init Binary: docker-init
 containerd version: 71909c1814c544ac47ab91d2e8b84718e517bb99
 runc version: v1.1.11-0-g4bccb38
 init version: de40ad0
 Security Options:
  seccomp
   Profile: builtin
  cgroupns
 Kernel Version: 6.5.11-linuxkit
 Operating System: Alpine Linux v3.19
 OSType: linux
 Architecture: aarch64
 CPUs: 10
 Total Memory: 7.663GiB
 Name: 4a7ed206a70d
 ID: c20f7230-59a2-4824-a2f4-fda71c982ee6
 Docker Root Dir: /var/lib/docker
 Debug Mode: false
 Experimental: false
 Insecure Registries:
  127.0.0.0/8
 Live Restore Enabled: false
 Product License: Community Engine
```

### <a name="format"></a> Format the output (--format)

You can also specify the output format:

```console
$ docker info --format '{{json .}}'

{"ID":"4cee4408-10d2-4e17-891c-a41736ac4536","Containers":14, ...}
```

### Run `docker info` on Windows

Here is a sample output for a daemon running on Windows Server:

```console
C:\> docker info

Client: Docker Engine - Community
 Version:    24.0.0
 Context:    default
 Debug Mode: false
 Plugins:
  buildx: Docker Buildx (Docker Inc.)
    Version:  v0.10.4
    Path:     C:\Program Files\Docker\cli-plugins\docker-buildx.exe
  compose: Docker Compose (Docker Inc.)
    Version:  v2.17.2
    Path:     C:\Program Files\Docker\cli-plugins\docker-compose.exe

Server:
 Containers: 1
  Running: 0
  Paused: 0
  Stopped: 1
 Images: 17
 Server Version: 23.0.3
 Storage Driver: windowsfilter
 Logging Driver: json-file
 Plugins:
  Volume: local
  Network: ics internal l2bridge l2tunnel nat null overlay private transparent
  Log: awslogs etwlogs fluentd gcplogs gelf json-file local splunk syslog
 Swarm: inactive
 Default Isolation: process
 Kernel Version: 10.0 20348 (20348.1.amd64fre.fe_release.210507-1500)
 Operating System: Microsoft Windows Server Version 21H2 (OS Build 20348.707)
 OSType: windows
 Architecture: x86_64
 CPUs: 8
 Total Memory: 3.999 GiB
 Name: WIN-V0V70C0LU5P
 ID: 2880d38d-464e-4d01-91bd-c76f33ba3981
 Docker Root Dir: C:\ProgramData\docker
 Debug Mode: false
 Experimental: true
 Insecure Registries:
  myregistry:5000
  127.0.0.0/8
 Registry Mirrors:
   http://192.168.1.2/
   http://registry-mirror.example.com:5000/
 Live Restore Enabled: false
```
