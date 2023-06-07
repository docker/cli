# start

<!---MARKER_GEN_START-->
Start one or more stopped containers

### Aliases

`docker container start`, `docker start`

### Options

| Name                  | Type     | Default | Description                                         |
|:----------------------|:---------|:--------|:----------------------------------------------------|
| `-a`, `--attach`      |          |         | Attach STDOUT/STDERR and forward signals            |
| `--checkpoint`        | `string` |         | Restore from this checkpoint                        |
| `--checkpoint-dir`    | `string` |         | Use a custom checkpoint storage directory           |
| `--detach-keys`       | `string` |         | Override the key sequence for detaching a container |
| `-i`, `--interactive` |          |         | Attach container's STDIN                            |


<!---MARKER_GEN_END-->

## Examples

### Start a container

To start a container, specify the container ID or image name. For example:

```console
$ docker start my_container
```

### <a name="attach"></a>  Start a container and attach to its STDOUT/STDERR (--attach, -a)

To start a container and attach to its STDOUT/STDERR and forward signals, use the `--attach` or `-a` option. For example, if you create an nginx container named my_nginx_container, you can start it and monitor its logs using the following:

```console
$ docker create --name my_nginx_container -p 80:80 nginx
ca5a4351c9c4b43c0b0c69d4d925aa1d8a53a3b33742250ad227f22096accab6

$ docker start -a my_nginx_container
/docker-entrypoint.sh: /docker-entrypoint.d/ is not empty, will attempt to perform configuration
/docker-entrypoint.sh: Looking for shell scripts in /docker-entrypoint.d/
/docker-entrypoint.sh: Launching /docker-entrypoint.d/10-listen-on-ipv6-by-default.sh
...
```

### <a name="checkpoint"></a> Start a container and restore from a checkpoint (--checkpoint) (experimental)

The `--checkpoint` option is an experimental feature, and should not be
considered stable. To read about experimental daemon options and how to enable
them, see
[Daemon configuration file](https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-configuration-file).

To start a container and restore it from a checkpoint, use the `--checkpoint` option. For example:

```console
$ docker run --security-opt=seccomp:unconfined --name cr -d busybox /bin/sh -c 'i=0; while true; do echo $i; i=$(expr $i + 1); sleep 1; done'
db316bda7d64b4154d207a3d659c90c982d0b35a3e177fc396e809a6a93a147f

$ docker checkpoint create cr checkpoint1
checkpoint1

# <later>
$ docker start --checkpoint checkpoint1 cr
```

### <a name="checkpoint-dir"></a>  Start a container and restore from a custom checkpoint storage (--checkpoint-dir) (experimental)

The `--checkpoint-dir` option is an experimental feature, and should not be
considered stable. To read about experimental daemon options and how to enable
them, see
[Daemon configuration file](https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-configuration-file).

To start a container and restore it from a checkpoint in a custom checkpoint storage directory, use the `--checkpoint-dir` option. For example:

```console
$ docker run --security-opt=seccomp:unconfined --name cr -d busybox /bin/sh -c 'i=0; while true; do echo $i; i=$(expr $i + 1); sleep 1; done'
db316bda7d64b4154d207a3d659c90c982d0b35a3e177fc396e809a6a93a147f

$ docker checkpoint create --checkpoint-dir /path/to/checkpoints cr checkpoint2
checkpoint2

# <later>
$ docker start --checkpoint-dir /path/to/checkpoints --checkpoint checkpoint2 cr
```

### <a name="detach-keys"></a> Start a container and override the detach key sequence (--detach-keys)

To start a container and override the key sequence for detaching a container, use the `--detach-keys` option. For example to change the detach sequence to `ctrl` plus `x`, use the following:

```console
$ docker start -a --detach-keys="ctrl-x" my_container
```

### <a name="interactive"></a> Start a container and attach to its STDIN (--interactive, -i)

To start a container and attach to its STDIN, use the `--interactive` or `-i` option. For example, if you create an ubuntu container named my_ubuntu_container, you can start it and interact with its shell using the following:

```console
$ docker create -it --name my_ubuntu_container ubuntu
6facdc392ec364d53d5cca760791d33b173d89525aff8f6f7c73a68bda0ab33c

$ docker start -i my_ubuntu_container
root@6facdc392ec3:/#
```