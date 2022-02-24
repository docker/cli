---
title: "version"
description: "The version command description and usage"
keywords: "version, architecture, api"
---

# version

```markdown
Usage:  docker version [OPTIONS]

Show the Docker version information

Options:
  -f, --format string       Format the output using the given Go template
      --help                Print usage
```

## Description

By default, this will render all version information in an easy to read
layout. If a format is specified, the given template will be executed instead.

Go's [text/template](https://golang.org/pkg/text/template/) package
describes all the details of the format.

## Examples

### Default output

```console
$ docker version

Client:
 Version:           19.03.8
 API version:       1.40
 Go version:        go1.12.17
 Git commit:        afacb8b
 Built:             Wed Mar 11 01:21:11 2020
 OS/Arch:           darwin/amd64
 Context:           default
 Experimental:      true

Server:
 Engine:
  Version:          19.03.8
  API version:      1.40 (minimum version 1.12)
  Go version:       go1.12.17
  Git commit:       afacb8b
  Built:            Wed Mar 11 01:29:16 2020
  OS/Arch:          linux/amd64
  Experimental:     true
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

### Get the server version

```console
$ docker version --format '{{.Server.Version}}'

19.03.8
```

### Dump raw JSON data

```console
$ docker version --format '{{json .}}'

{"Client":{"Platform":{"Name":"Docker Engine - Community"},"Version":"19.03.8","ApiVersion":"1.40","DefaultAPIVersion":"1.40","GitCommit":"afacb8b","GoVersion":"go1.12.17","Os":"darwin","Arch":"amd64","BuildTime":"Wed Mar 11 01:21:11 2020","Experimental":true},"Server":{"Platform":{"Name":"Docker Engine - Community"},"Components":[{"Name":"Engine","Version":"19.03.8","Details":{"ApiVersion":"1.40","Arch":"amd64","BuildTime":"Wed Mar 11 01:29:16 2020","Experimental":"true","GitCommit":"afacb8b","GoVersion":"go1.12.17","KernelVersion":"4.19.76-linuxkit","MinAPIVersion":"1.12","Os":"linux"}},{"Name":"containerd","Version":"v1.2.13","Details":{"GitCommit":"7ad184331fa3e55e52b890ea95e65ba581ae3429"}},{"Name":"runc","Version":"1.0.0-rc10","Details":{"GitCommit":"dc9208a3303feef5b3839f4323d9beb36df0a9dd"}},{"Name":"docker-init","Version":"0.18.0","Details":{"GitCommit":"fec3683"}}],"Version":"19.03.8","ApiVersion":"1.40","MinAPIVersion":"1.12","GitCommit":"afacb8b","GoVersion":"go1.12.17","Os":"linux","Arch":"amd64","KernelVersion":"4.19.76-linuxkit","Experimental":true,"BuildTime":"2020-03-11T01:29:16.000000000+00:00"}}
```

### Print the current context

The following example prints the currently used [`docker context`](context.md):

```console
$ docker version --format='{{.Client.Context}}'
default
```

As an example, this output can be used to dynamically change your shell prompt
to indicate your active context. The example below illustrates how this output
could be used when using Bash as your shell.

Declare a function to obtain the current context in your `~/.bashrc`, and set
this command as your `PROMPT_COMMAND`

```console
function docker_context_prompt() {
        PS1="context: $(docker version --format='{{.Client.Context}}')> "
}

PROMPT_COMMAND=docker_context_prompt
```

After reloading the `~/.bashrc`, the prompt now shows the currently selected
`docker context`:

```console
$ source ~/.bashrc
context: default> docker context create --docker host=unix:///var/run/docker.sock my-context
my-context
Successfully created context "my-context"
context: default> docker context use my-context
my-context
Current context is now "my-context"
context: my-context> docker context use default
default
Current context is now "default"
context: default>
```

Refer to the [`docker context` section](context.md) in the command line reference
for more information about `docker context`.
