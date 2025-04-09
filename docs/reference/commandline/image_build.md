# build (legacy builder)

<!---MARKER_GEN_START-->
Build an image from a Dockerfile

### Aliases

`docker image build`, `docker build`, `docker builder build`

### Options

| Name                                                                                                                                                 | Type          | Default   | Description                                                       |
|:-----------------------------------------------------------------------------------------------------------------------------------------------------|:--------------|:----------|:------------------------------------------------------------------|
| [`--add-host`](https://docs.docker.com/reference/cli/docker/buildx/build/#add-host)                                                                  | `list`        |           | Add a custom host-to-IP mapping (`host:ip`)                       |
| [`--build-arg`](https://docs.docker.com/reference/cli/docker/buildx/build/#build-arg)                                                                | `list`        |           | Set build-time variables                                          |
| `--cache-from`                                                                                                                                       | `stringSlice` |           | Images to consider as cache sources                               |
| [`--cgroup-parent`](https://docs.docker.com/reference/cli/docker/buildx/build/#cgroup-parent)                                                        | `string`      |           | Set the parent cgroup for the `RUN` instructions during build     |
| `--compress`                                                                                                                                         | `bool`        |           | Compress the build context using gzip                             |
| `--cpu-period`                                                                                                                                       | `int64`       | `0`       | Limit the CPU CFS (Completely Fair Scheduler) period              |
| `--cpu-quota`                                                                                                                                        | `int64`       | `0`       | Limit the CPU CFS (Completely Fair Scheduler) quota               |
| `-c`, `--cpu-shares`                                                                                                                                 | `int64`       | `0`       | CPU shares (relative weight)                                      |
| `--cpuset-cpus`                                                                                                                                      | `string`      |           | CPUs in which to allow execution (0-3, 0,1)                       |
| `--cpuset-mems`                                                                                                                                      | `string`      |           | MEMs in which to allow execution (0-3, 0,1)                       |
| `--disable-content-trust`                                                                                                                            | `bool`        | `true`    | Skip image verification                                           |
| [`-f`](https://docs.docker.com/reference/cli/docker/buildx/build/#file), [`--file`](https://docs.docker.com/reference/cli/docker/buildx/build/#file) | `string`      |           | Name of the Dockerfile (Default is `PATH/Dockerfile`)             |
| `--force-rm`                                                                                                                                         | `bool`        |           | Always remove intermediate containers                             |
| `--iidfile`                                                                                                                                          | `string`      |           | Write the image ID to the file                                    |
| [`--isolation`](#isolation)                                                                                                                          | `string`      |           | Container isolation technology                                    |
| `--label`                                                                                                                                            | `list`        |           | Set metadata for an image                                         |
| `-m`, `--memory`                                                                                                                                     | `bytes`       | `0`       | Memory limit                                                      |
| `--memory-swap`                                                                                                                                      | `bytes`       | `0`       | Swap limit equal to memory plus swap: -1 to enable unlimited swap |
| [`--network`](https://docs.docker.com/reference/cli/docker/buildx/build/#network)                                                                    | `string`      | `default` | Set the networking mode for the RUN instructions during build     |
| `--no-cache`                                                                                                                                         | `bool`        |           | Do not use cache when building the image                          |
| `--platform`                                                                                                                                         | `string`      |           | Set platform if server is multi-platform capable                  |
| `--pull`                                                                                                                                             | `bool`        |           | Always attempt to pull a newer version of the image               |
| `-q`, `--quiet`                                                                                                                                      | `bool`        |           | Suppress the build output and print image ID on success           |
| `--rm`                                                                                                                                               | `bool`        | `true`    | Remove intermediate containers after a successful build           |
| [`--security-opt`](#security-opt)                                                                                                                    | `stringSlice` |           | Security options                                                  |
| `--shm-size`                                                                                                                                         | `bytes`       | `0`       | Size of `/dev/shm`                                                |
| [`--squash`](#squash)                                                                                                                                | `bool`        |           | Squash newly built layers into a single new layer                 |
| [`-t`](https://docs.docker.com/reference/cli/docker/buildx/build/#tag), [`--tag`](https://docs.docker.com/reference/cli/docker/buildx/build/#tag)    | `list`        |           | Name and optionally a tag in the `name:tag` format                |
| [`--target`](https://docs.docker.com/reference/cli/docker/buildx/build/#target)                                                                      | `string`      |           | Set the target build stage to build.                              |
| `--ulimit`                                                                                                                                           | `ulimit`      |           | Ulimit options                                                    |


<!---MARKER_GEN_END-->

## Description

> [!IMPORTANT]
> This page refers to the **legacy implementation** of `docker build`,
> using the legacy (pre-BuildKit) build backend.
> This configuration is only relevant if you're building Windows containers.
>
> For information about the default `docker build`, using Buildx,
> see [`docker buildx build`](https://docs.docker.com/reference/cli/docker/build/).

When building with legacy builder, images are created from a Dockerfile by
running a sequence of [commits](./container_commit.md). This process is
inefficient and slow compared to using BuildKit, which is why this build
strategy is deprecated for all use cases except for building Windows
containers. It's still useful for building Windows containers because BuildKit
doesn't yet have full feature parity for Windows.

Builds invoked with `docker build` use Buildx (and BuildKit) by default, unless:

- You're running Docker Engine in Windows container mode
- You explicitly opt out of using BuildKit by setting the environment variable `DOCKER_BUILDKIT=0`.

The descriptions on this page only covers information that's exclusive to the
legacy builder, and cases where behavior in the legacy builder deviates from
behavior in BuildKit. For information about features and flags that are common
between the legacy builder and BuildKit, such as `--tag` and `--target`, refer
to the documentation for [`docker buildx build`](https://docs.docker.com/reference/cli/docker/buildx/build/).

### Build context with the legacy builder

The build context is the positional argument you pass when invoking the build
command. In the following example, the context is `.`, meaning current the
working directory.

```console
$ docker build .
```

When using the legacy builder, the build context is sent over to the daemon in
its entirety. With BuildKit, only the files you use in your builds are
transmitted. The legacy builder doesn't calculate which files it needs
beforehand. This means that for builds with a large context, context transfer
can take a long time, even if you're only using a subset of the files included
in the context.

When using the legacy builder, it's therefore extra important that you
carefully consider what files you include in the context you specify. Use a
[`.dockerignore`](https://docs.docker.com/build/concepts/context/#dockerignore-files)
file to exclude files and directories that you don't require in your build from
being sent as part of the build context.

#### Accessing paths outside the build context

The legacy builder will error out if you try to access files outside of the
build context using relative paths in your Dockerfile.

```dockerfile
FROM alpine
COPY ../../some-dir .
```

```console
$ docker build .
...
Step 2/2 : COPY ../../some-dir .
COPY failed: forbidden path outside the build context: ../../some-dir ()
```

BuildKit on the other hand strips leading relative paths that traverse outside
of the build context. Re-using the previous example, the path `COPY
../../some-dir .` evaluates to `COPY some-dir .` with BuildKit.

## Examples

### <a name="isolation"></a> Specify isolation technology for container (--isolation)

This option is useful in situations where you are running Docker containers on
Windows. The `--isolation=<value>` option sets a container's isolation
technology. On Linux, the only supported is the `default` option which uses
Linux namespaces. On Microsoft Windows, you can specify these values:


| Value     | Description                                                                                                                                                                    |
|-----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `default` | Use the value specified by the Docker daemon's `--exec-opt` . If the `daemon` does not specify an isolation technology, Microsoft Windows uses `process` as its default value. |
| `process` | Namespace isolation only.                                                                                                                                                      |
| `hyperv`  | Hyper-V hypervisor partition-based isolation.                                                                                                                                  |

### <a name="security-opt"></a> Optional security options (--security-opt)

This flag is only supported on a daemon running on Windows, and only supports
the `credentialspec` option. The `credentialspec` must be in the format
`file://spec.txt` or `registry://keyname`.

### <a name="squash"></a> Squash an image's layers (--squash) (experimental)

#### Overview

> [!NOTE]
> The `--squash` option is an experimental feature, and should not be considered
> stable.

Once the image is built, this flag squashes the new layers into a new image with
a single new layer. Squashing doesn't destroy any existing image, rather it
creates a new image with the content of the squashed layers. This effectively
makes it look like all `Dockerfile` commands were created with a single layer.
The `--squash` flag preserves the build cache.

Squashing layers can be beneficial if your Dockerfile produces multiple layers
modifying the same files. For example, files created in one step and
removed in another step. For other use-cases, squashing images may actually have
a negative impact on performance. When pulling an image consisting of multiple
layers, the daemon can pull layers in parallel and allows sharing layers between
images (saving space).

For most use cases, multi-stage builds are a better alternative, as they give more
fine-grained control over your build, and can take advantage of future
optimizations in the builder. Refer to the [Multi-stage builds](https://docs.docker.com/build/building/multi-stage/)
section for more information.

#### Known limitations

The `--squash` option has a number of known limitations:

- When squashing layers, the resulting image can't take advantage of layer
  sharing with other images, and may use significantly more space. Sharing the
  base image is still supported.
- When using this option you may see significantly more space used due to
  storing two copies of the image, one for the build cache with all the cache
  layers intact, and one for the squashed version.
- While squashing layers may produce smaller images, it may have a negative
  impact on performance, as a single layer takes longer to extract, and
  you can't parallelize downloading a single layer.
- When attempting to squash an image that doesn't make changes to the
  filesystem (for example, the Dockerfile only contains `ENV` instructions),
  the squash step will fail (see [issue #33823](https://github.com/moby/moby/issues/33823)).

#### Prerequisites

The example on this page is using experimental mode in Docker 23.03.

You can enable experimental mode by using the `--experimental` flag when starting
the Docker daemon or setting `experimental: true` in the `daemon.json` configuration
file.

By default, experimental mode is disabled. To see the current configuration of
the Docker daemon, use the `docker version` command and check the `Experimental`
line in the `Engine` section:

```console
Client: Docker Engine - Community
 Version:           23.0.3
 API version:       1.42
 Go version:        go1.19.7
 Git commit:        3e7cbfd
 Built:             Tue Apr  4 22:05:41 2023
 OS/Arch:           darwin/amd64
 Context:           default

Server: Docker Engine - Community
 Engine:
  Version:          23.0.3
  API version:      1.42 (minimum version 1.12)
  Go version:       go1.19.7
  Git commit:       59118bf
  Built:            Tue Apr  4 22:05:41 2023
  OS/Arch:          linux/amd64
  Experimental:     true
 [...]
```

#### Build an image with the `--squash` flag

The following is an example of a build with the `--squash` flag.  Below is the
`Dockerfile`:

```dockerfile
FROM busybox
RUN echo hello > /hello
RUN echo world >> /hello
RUN touch remove_me /remove_me
ENV HELLO=world
RUN rm /remove_me
```

Next, build an image named `test` using the `--squash` flag.

```console
$ docker build --squash -t test .
```

After the build completes, the history looks like the below. The history could show that a layer's
name is `<missing>`, and there is a new layer with COMMENT `merge`.

```console
$ docker history test

IMAGE               CREATED             CREATED BY                                      SIZE                COMMENT
4e10cb5b4cac        3 seconds ago                                                       12 B                merge sha256:88a7b0112a41826885df0e7072698006ee8f621c6ab99fca7fe9151d7b599702 to sha256:47bcc53f74dc94b1920f0b34f6036096526296767650f223433fe65c35f149eb
<missing>           5 minutes ago       /bin/sh -c rm /remove_me                        0 B
<missing>           5 minutes ago       /bin/sh -c #(nop) ENV HELLO=world               0 B
<missing>           5 minutes ago       /bin/sh -c touch remove_me /remove_me           0 B
<missing>           5 minutes ago       /bin/sh -c echo world >> /hello                 0 B
<missing>           6 minutes ago       /bin/sh -c echo hello > /hello                  0 B
<missing>           7 weeks ago         /bin/sh -c #(nop) CMD ["sh"]                    0 B
<missing>           7 weeks ago         /bin/sh -c #(nop) ADD file:47ca6e777c36a4cfff   1.113 MB
```

Test the image, check for `/remove_me` being gone, make sure `hello\nworld` is
in `/hello`, make sure the `HELLO` environment variable's value is `world`.
