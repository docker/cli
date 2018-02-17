---
title: "exec"
description: "The exec command description and usage"
keywords: "command, container, run, execute"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# exec

```markdown
Usage:  docker exec [OPTIONS] CONTAINER COMMAND [ARG...]

Run a command in a running container

Options:
  -d, --detach         Detached mode: run command in the background
      --detach-keys    Override the key sequence for detaching a container
  -e, --env=[]         Set environment variables
      --help           Print usage
  -i, --interactive    Keep STDIN open even if not attached
      --privileged     Give extended privileges to the command
  -t, --tty            Allocate a pseudo-TTY
  -u, --user           Username or UID (format: <name|uid>[:<group|gid>])
  -w, --workdir        Working directory inside the container  
```

## Description

The `docker exec` command runs a new command in a running container.

The command started using `docker exec` only runs while the container's primary
process (`PID 1`) is running, and it is not restarted if the container is
restarted.

COMMAND will run in the default directory of the container. If the
underlying image has a custom directory specified with the WORKDIR directive
in its Dockerfile, this will be used instead.

COMMAND should be an executable, a chained or a quoted command
will not work. Example: `docker exec -ti my_container "echo a && echo b"` will
not work, but `docker exec -ti my_container sh -c "echo a && echo b"` will.

## Examples

### Run `docker exec` on a running container

First, start a container.

```bash
$ docker run --name ubuntu_bash --rm -i -t ubuntu bash
```

This will create a container named `ubuntu_bash` and start a Bash session.

Next, execute a command on the container.

```bash
$ docker exec -d ubuntu_bash touch /tmp/execWorks
```

This will create a new file `/tmp/execWorks` inside the running container
`ubuntu_bash`, in the background.

Next, execute an interactive `bash` shell on the container.

```bash
$ docker exec -it ubuntu_bash bash
```

This will create a new Bash session in the container `ubuntu_bash`.

Next, set an environment variable in the current bash session.

```bash
$ docker exec -it -e VAR=1 ubuntu_bash bash
```

This will create a new Bash session in the container `ubuntu_bash` with environment 
variable `$VAR` set to "1". Note that this environment variable will only be valid 
on the current Bash session.

By default `docker exec` command runs in the same working directory set when container was created.

```bash
$ docker exec -it ubuntu_bash pwd
/
```

You can select working directory for the command to execute into

```bash
$ docker exec -it -w /root ubuntu_bash pwd
/root
```


### Try to run `docker exec` on a paused container

If the container is paused, then the `docker exec` command will fail with an error:

```bash
$ docker pause test

test

$ docker ps

CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS                   PORTS               NAMES
1ae3b36715d2        ubuntu:latest       "bash"              17 seconds ago      Up 16 seconds (Paused)                       test

$ docker exec test ls

FATA[0000] Error response from daemon: Container test is paused, unpause the container before exec

$ echo $?
1
```
