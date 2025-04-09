# commit

<!---MARKER_GEN_START-->
Create a new image from a container's changes

### Aliases

`docker container commit`, `docker commit`

### Options

| Name                                   | Type     | Default | Description                                                |
|:---------------------------------------|:---------|:--------|:-----------------------------------------------------------|
| `-a`, `--author`                       | `string` |         | Author (e.g., `John Hannibal Smith <hannibal@a-team.com>`) |
| [`-c`](#change), [`--change`](#change) | `list`   |         | Apply Dockerfile instruction to the created image          |
| `-m`, `--message`                      | `string` |         | Commit message                                             |
| `-p`, `--pause`                        | `bool`   | `true`  | Pause container during commit                              |


<!---MARKER_GEN_END-->

## Description

It can be useful to commit a container's file changes or settings into a new
image. This lets you debug a container by running an interactive shell, or
export a working dataset to another server.

Commits do not include any data contained in mounted volumes.

By default, the container being committed and its processes will be paused
while the image is committed. This reduces the likelihood of encountering data
corruption during the process of creating the commit. If this behavior is
undesired, set the `--pause` option to false.

The `--change` option will apply `Dockerfile` instructions to the image that's
created. Supported `Dockerfile` instructions:
`CMD`|`ENTRYPOINT`|`ENV`|`EXPOSE`|`LABEL`|`ONBUILD`|`USER`|`VOLUME`|`WORKDIR`

## Examples

### Commit a container

```console
$ docker ps

CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS              NAMES
c3f279d17e0a        ubuntu:24.04        /bin/bash           7 days ago          Up 25 hours                            desperate_dubinsky
197387f1b436        ubuntu:24.04        /bin/bash           7 days ago          Up 25 hours                            focused_hamilton

$ docker commit c3f279d17e0a  svendowideit/testimage:version3

f5283438590d

$ docker images

REPOSITORY                        TAG                 ID                  CREATED             SIZE
svendowideit/testimage            version3            f5283438590d        16 seconds ago      335.7 MB
```

### <a name="change"></a> Commit a container with new configurations (--change)

```console
$ docker ps

CONTAINER ID       IMAGE               COMMAND             CREATED             STATUS              PORTS              NAMES
c3f279d17e0a       ubuntu:24.04        /bin/bash           7 days ago          Up 25 hours                            desperate_dubinsky
197387f1b436       ubuntu:24.04        /bin/bash           7 days ago          Up 25 hours                            focused_hamilton

$ docker inspect -f "{{ .Config.Env }}" c3f279d17e0a

[HOME=/ PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin]

$ docker commit --change "ENV DEBUG=true" c3f279d17e0a  svendowideit/testimage:version3

f5283438590d

$ docker inspect -f "{{ .Config.Env }}" f5283438590d

[HOME=/ PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin DEBUG=true]
```

### Commit a container with new `CMD` and `EXPOSE` instructions

```console
$ docker ps

CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS              NAMES
c3f279d17e0a        ubuntu:24.04        /bin/bash           7 days ago          Up 25 hours                            desperate_dubinsky
197387f1b436        ubuntu:24.04        /bin/bash           7 days ago          Up 25 hours                            focused_hamilton

$ docker commit --change='CMD ["apachectl", "-DFOREGROUND"]' -c "EXPOSE 80" c3f279d17e0a  svendowideit/testimage:version4

f5283438590d

$ docker run -d svendowideit/testimage:version4

89373736e2e7f00bc149bd783073ac43d0507da250e999f3f1036e0db60817c0

$ docker ps

CONTAINER ID        IMAGE               COMMAND                 CREATED             STATUS              PORTS              NAMES
89373736e2e7        testimage:version4  "apachectl -DFOREGROU"  3 seconds ago       Up 2 seconds        80/tcp             distracted_fermat
c3f279d17e0a        ubuntu:24.04        /bin/bash               7 days ago          Up 25 hours                            desperate_dubinsky
197387f1b436        ubuntu:24.04        /bin/bash               7 days ago          Up 25 hours                            focused_hamilton
```
