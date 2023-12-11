# exec

<!---MARKER_GEN_START-->
Execute a command in a running container

### Aliases

`docker container exec`, `docker exec`

### Options

| Name                                      | Type     | Default | Description                                            |
|:------------------------------------------|:---------|:--------|:-------------------------------------------------------|
| `-d`, `--detach`                          |          |         | Detached mode: run command in the background           |
| `--detach-keys`                           | `string` |         | Override the key sequence for detaching a container    |
| [`-e`](#env), [`--env`](#env)             | `list`   |         | Set environment variables                              |
| `--env-file`                              | `list`   |         | Read in a file of environment variables                |
| `-i`, `--interactive`                     |          |         | Keep STDIN open even if not attached                   |
| `--privileged`                            |          |         | Give extended privileges to the command                |
| `-t`, `--tty`                             |          |         | Allocate a pseudo-TTY                                  |
| `-u`, `--user`                            | `string` |         | Username or UID (format: `<name\|uid>[:<group\|gid>]`) |
| [`-w`](#workdir), [`--workdir`](#workdir) | `string` |         | Working directory inside the container                 |


<!---MARKER_GEN_END-->

## Description

The `docker exec` command runs a new command in a running container.

The command started using `docker exec` only runs while the container's primary
process (`PID 1`) is running, and it is not restarted if the container is
restarted.

COMMAND runs in the default directory of the container. If the underlying image
has a custom directory specified with the WORKDIR directive in its Dockerfile,
this directory is used instead.

COMMAND must be an executable. A chained or a quoted command does not work.
For example, `docker exec -it my_container sh -c "echo a && echo b"` does
work, but `docker exec -it my_container "echo a && echo b"` does not.

## Examples

### Run `docker exec` on a running container

First, start a container.

```console
$ docker run --name mycontainer -d -i -t alpine /bin/sh
```

This creates and starts a container named `mycontainer` from an `alpine` image
with an `sh` shell as its main process. The `-d` option (shorthand for `--detach`)
sets the container to run in the background, in detached mode, with a pseudo-TTY
attached (`-t`). The `-i` option is set to keep `STDIN` attached (`-i`), which
prevents the `sh` process from exiting immediately.

Next, execute a command on the container.

```console
$ docker exec -d mycontainer touch /tmp/execWorks
```

This creates a new file `/tmp/execWorks` inside the running container
`mycontainer`, in the background.

Next, execute an interactive `sh` shell on the container.

```console
$ docker exec -it mycontainer sh
```

This starts a new shell session in the container `mycontainer`.

### <a name="env"></a> Set environment variables for the exec process (--env, -e)

Next, set environment variables in the current bash session.

The `docker exec` command inherits the environment variables that are set at the
time the container is created. Use the `--env` (or the `-e` shorthand) to
override global environment variables, or to set additional environment
variables for the process started by `docker exec`.

The example below creates a new shell session in the container `mycontainer` with
environment variables `$VAR_A` and `$VAR_B` set to "1" and "2" respectively.
These environment variables are only valid for the `sh` process started by that
`docker exec` command, and are not available to other processes running inside
the container.

```console
$ docker exec -e VAR_A=1 -e VAR_B=2 mycontainer env
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
HOSTNAME=f64a4851eb71
VAR_A=1
VAR_B=2
HOME=/root
```

### <a name="workdir"></a> Set the working directory for the exec process (--workdir, -w)

By default `docker exec` command runs in the same working directory set when 
the container was created.

```console
$ docker exec -it mycontainer pwd
/
```

You can specify an alternative working directory for the command to execute 
using the `--workdir` option (or the `-w` shorthand):

```console
$ docker exec -it -w /root mycontainer pwd
/root
```


### Try to run `docker exec` on a paused container

If the container is paused, then the `docker exec` command fails with an error:

```console
$ docker pause mycontainer
mycontainer

$ docker ps

CONTAINER ID   IMAGE     COMMAND     CREATED          STATUS                   PORTS     NAMES
482efdf39fac   alpine    "/bin/sh"   17 seconds ago   Up 16 seconds (Paused)             mycontainer

$ docker exec mycontainer sh

Error response from daemon: Container mycontainer is paused, unpause the container before exec

$ echo $?
1
```
