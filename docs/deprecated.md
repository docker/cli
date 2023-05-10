---
aliases: ["/engine/misc/deprecated/"]
description: "Deprecated Features."
keywords: "docker, documentation, about, technology, deprecate"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/docker/cli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# Deprecated Engine Features

This page provides an overview of features that are deprecated in Engine. Changes
in packaging, and supported (Linux) distributions are not included. To learn
about end of support for Linux distributions, refer to the
[release notes](https://docs.docker.com/engine/release-notes/).

## Feature Deprecation Policy

As changes are made to Docker there may be times when existing features need to
be removed or replaced with newer features. Before an existing feature is removed
it is labeled as "deprecated" within the documentation and remains in Docker for
at least one stable release unless specified explicitly otherwise. After that time
it may be removed.

Users are expected to take note of the list of deprecated features each release
and plan their migration away from those features, and (if applicable) towards
the replacement features as soon as possible.

## Deprecated Engine Features

The table below provides an overview of the current status of deprecated features:

- **Deprecated**: the feature is marked "deprecated" and should no longer be used.
  The feature may be removed, disabled, or change behavior in a future release.
  The _"Deprecated"_ column contains the release in which the feature was marked 
  deprecated, whereas the _"Remove"_ column contains a tentative release in which
  the feature is to be removed. If no release is included in the _"Remove"_ column,
  the release is yet to be decided on.
- **Removed**: the feature was removed, disabled, or hidden. Refer to the linked
  section for details. Some features are "soft" deprecated, which means that they
  remain functional for backward compatibility, and to allow users to migrate to
  alternatives. In such cases, a warning may be printed, and users should not rely
  on this feature.

| Status     | Feature                                                                                                                            | Deprecated | Remove  |
|------------|------------------------------------------------------------------------------------------------------------------------------------|------------|---------|
| Deprecated | [OOM-score adjust for the daemon](#oom-score-adjust-for-the-daemon)                                                                | v24.0.0    | v25.0.0 |
| Removed    | [Buildkit build information](#buildkit-build-information)                                                                          | v23.0.0    | v24.0.0 |
| Deprecated | [Legacy builder for Linux images](#legacy-builder-for-linux-images)                                                                | v23.0.0    | -       |
| Deprecated | [Legacy builder fallback](#legacy-builder-fallback)                                                                                | v23.0.0    | -       |
| Removed    | [Btrfs storage driver on CentOS 7 and RHEL 7](#btrfs-storage-driver-on-centos-7-and-rhel-7)                                        | v20.10     | v23.0.0 |
| Removed    | [Support for encrypted TLS private keys](#support-for-encrypted-tls-private-keys)                                                  | v20.10     | v23.0.0 |
| Removed    | [Kubernetes stack and context support](#kubernetes-stack-and-context-support)                                                      | v20.10     | v23.0.0 |
| Deprecated | [Pulling images from non-compliant image registries](#pulling-images-from-non-compliant-image-registries)                          | v20.10     | -       |
| Removed    | [Linux containers on Windows (LCOW)](#linux-containers-on-windows-lcow-experimental)                                               | v20.10     | v23.0.0 |
| Deprecated | [BLKIO weight options with cgroups v1](#blkio-weight-options-with-cgroups-v1)                                                      | v20.10     | -       |
| Removed    | [Kernel memory limit](#kernel-memory-limit)                                                                                        | v20.10     | v23.0.0 |
| Removed    | [Classic Swarm and overlay networks using external key/value stores](#classic-swarm-and-overlay-networks-using-cluster-store)      | v20.10     | v23.0.0 |
| Removed    | [Support for the legacy `~/.dockercfg` configuration file for authentication](#support-for-legacy-dockercfg-configuration-files)   | v20.10     | v23.0.0 |
| Deprecated | [CLI plugins support](#cli-plugins-support)                                                                                        | v20.10     | -       |
| Deprecated | [Dockerfile legacy `ENV name value` syntax](#dockerfile-legacy-env-name-value-syntax)                                              | v20.10     | -       |
| Removed    | [`docker build --stream` flag (experimental)](#docker-build---stream-flag-experimental)                                            | v20.10     | v20.10  |
| Deprecated | [`fluentd-async-connect` log opt](#fluentd-async-connect-log-opt)                                                                  | v20.10     | -       |
| Removed    | [Configuration options for experimental CLI features](#configuration-options-for-experimental-cli-features)                        | v19.03     | v23.0.0 |
| Deprecated | [Pushing and pulling with image manifest v2 schema 1](#pushing-and-pulling-with-image-manifest-v2-schema-1)                        | v19.03     | v20.10  |
| Removed    | [`docker engine` subcommands](#docker-engine-subcommands)                                                                          | v19.03     | v20.10  |
| Removed    | [Top-level `docker deploy` subcommand (experimental)](#top-level-docker-deploy-subcommand-experimental)                            | v19.03     | v20.10  |
| Removed    | [`docker stack deploy` using "dab" files (experimental)](#docker-stack-deploy-using-dab-files-experimental)                        | v19.03     | v20.10  |
| Disabled   | [Support for the `overlay2.override_kernel_check` storage option](#support-for-the-overlay2override_kernel_check-storage-option)   | v19.03     | -       |
| Disabled   | [AuFS storage driver](#aufs-storage-driver)                                                                                        | v19.03     | -       |
| Disabled   | [Legacy "overlay" storage driver](#legacy-overlay-storage-driver)                                                                  | v18.09     | -       |
| Disabled   | [Device mapper storage driver](#device-mapper-storage-driver)                                                                      | v18.09     | -       |
| Removed    | [Use of reserved namespaces in engine labels](#use-of-reserved-namespaces-in-engine-labels)                                        | v18.06     | v20.10  |
| Removed    | [`--disable-legacy-registry` override daemon option](#--disable-legacy-registry-override-daemon-option)                            | v17.12     | v19.03  |
| Removed    | [Interacting with V1 registries](#interacting-with-v1-registries)                                                                  | v17.06     | v17.12  |
| Removed    | [Asynchronous `service create` and `service update` as default](#asynchronous-service-create-and-service-update-as-default)        | v17.05     | v17.10  |
| Removed    | [`-g` and `--graph` flags on `dockerd`](#-g-and---graph-flags-on-dockerd)                                                          | v17.05     | -       |
| Deprecated | [Top-level network properties in NetworkSettings](#top-level-network-properties-in-networksettings)                                | v1.13      | v17.12  |
| Removed    | [`filter` param for `/images/json` endpoint](#filter-param-for-imagesjson-endpoint)                                                | v1.13      | v20.10  |
| Removed    | [`repository:shortid` image references](#repositoryshortid-image-references)                                                       | v1.13      | v17.12  |
| Removed    | [`docker daemon` subcommand](#docker-daemon-subcommand)                                                                            | v1.13      | v17.12  |
| Removed    | [Duplicate keys with conflicting values in engine labels](#duplicate-keys-with-conflicting-values-in-engine-labels)                | v1.13      | v17.12  |
| Deprecated | [`MAINTAINER` in Dockerfile](#maintainer-in-dockerfile)                                                                            | v1.13      | -       |
| Deprecated | [API calls without a version](#api-calls-without-a-version)                                                                        | v1.13      | v17.12  |
| Removed    | [Backing filesystem without `d_type` support for overlay/overlay2](#backing-filesystem-without-d_type-support-for-overlayoverlay2) | v1.13      | v17.12  |
| Removed    | [`--automated` and `--stars` flags on `docker search`](#--automated-and---stars-flags-on-docker-search)                            | v1.12      | v20.10  |
| Deprecated | [`-h` shorthand for `--help`](#-h-shorthand-for---help)                                                                            | v1.12      | v17.09  |
| Removed    | [`-e` and `--email` flags on `docker login`](#-e-and---email-flags-on-docker-login)                                                | v1.11      | v17.06  |
| Deprecated | [Separator (`:`) of `--security-opt` flag on `docker run`](#separator--of---security-opt-flag-on-docker-run)                       | v1.11      | v17.06  |
| Deprecated | [Ambiguous event fields in API](#ambiguous-event-fields-in-api)                                                                    | v1.10      | -       |
| Removed    | [`-f` flag on `docker tag`](#-f-flag-on-docker-tag)                                                                                | v1.10      | v1.12   |
| Removed    | [HostConfig at API container start](#hostconfig-at-api-container-start)                                                            | v1.10      | v1.12   |
| Removed    | [`--before` and `--since` flags on `docker ps`](#--before-and---since-flags-on-docker-ps)                                          | v1.10      | v1.12   |
| Removed    | [Driver-specific log tags](#driver-specific-log-tags)                                                                              | v1.9       | v1.12   |
| Removed    | [Docker Content Trust `ENV` passphrase variables name change](#docker-content-trust-env-passphrase-variables-name-change)          | v1.9       | v1.12   |
| Removed    | [`/containers/(id or name)/copy` endpoint](#containersid-or-namecopy-endpoint)                                                     | v1.8       | v1.12   |
| Removed    | [LXC built-in exec driver](#lxc-built-in-exec-driver)                                                                              | v1.8       | v1.10   |
| Removed    | [Old Command Line Options](#old-command-line-options)                                                                              | v1.8       | v1.10   |
| Removed    | [`--api-enable-cors` flag on `dockerd`](#--api-enable-cors-flag-on-dockerd)                                                        | v1.6       | v17.09  |
| Removed    | [`--run` flag on `docker commit`](#--run-flag-on-docker-commit)                                                                    | v0.10      | v1.13   |
| Removed    | [Three arguments form in `docker import`](#three-arguments-form-in-docker-import)                                                  | v0.6.7     | v1.12   |

### OOM-score adjust for the daemon

**Deprecated in Release: v24.0.0**
**Target For Removal In Release: v25.0.0**

The `oom-score-adjust` option was added to prevent the daemon from being
OOM-killed before other processes. This option was mostly added as a
convenience, as running the daemon as a systemd unit was not yet common.

Having the daemon set its own limits is not best-practice, and something
better handled by the process-manager starting the daemon.

Docker v20.10 and newer no longer adjust the daemon's OOM score by default,
instead setting the OOM-score to the systemd unit (OOMScoreAdjust) that's
shipped with the packages.

Users currently depending on this feature are recommended to adjust the
daemon's OOM score using systemd or through other means, when starting
the daemon.

### Buildkit build information

**Deprecated in Release: v23.0.0**
**Removed in Release: v24.0.0**

[Build information](https://github.com/moby/buildkit/blob/v0.11/docs/buildinfo.md)
structures have been introduced in [BuildKit v0.10.0](https://github.com/moby/buildkit/releases/tag/v0.10.0)
and are generated with build metadata that allows you to see all the sources
(images, git repositories) that were used by the build with their exact
versions and also the configuration that was passed to the build. This
information is also embedded into the image configuration if one is generated.

### Legacy builder for Linux images

**Deprecated in Release: v23.0.0**

Docker v23.0.0 now uses BuildKit by default to build Linux images, and uses the
[Buildx](https://docs.docker.com/buildx/working-with-buildx/) CLI component for
`docker build`. With this change, `docker build` now exposes all advanced features
that BuildKit provides and which were previously only available through the
`docker buildx` subcommands.

The Buildx component is installed automatically when installing the `docker` CLI
using our `.deb` or `.rpm` packages, and statically linked binaries are provided
both on `download.docker.com`, and through the [`docker/buildx-bin` image](https://hub.docker.com/r/docker/buildx-bin)
on Docker Hub. Refer the [Buildx section](http://docs.docker.com/go/buildx/) for
detailed instructions on installing the Buildx component.

This release marks the beginning of the deprecation cycle of the classic ("legacy")
builder for Linux images. No active development will happen on the classic builder
(except for bugfixes). BuildKit development started five Years ago, left the
"experimental" phase since Docker 18.09, and is already the default builder for
[Docker Desktop](https://docs.docker.com/desktop/previous-versions/3.x-mac/#docker-desktop-320).
While we're comfortable that BuildKit is stable for general use, there may be
some changes in behavior. If you encounter issues with BuildKit, we encourage
you to report issues in the [BuildKit issue tracker on GitHub](https://github.com/moby/buildkit/){:target="_blank" rel="noopener" class="_"}

> Classic builder for building Windows images
>
> BuildKit does not (yet) provide support for building Windows images, and
> `docker build` continues to use the classic builder to build native Windows
> images on Windows daemons.

### Legacy builder fallback 

**Deprecated in Release: v23.0.0**

[Docker v23.0.0 now uses BuildKit by default to build Linux images](#legacy-builder-for-linux-images),
which requires the Buildx component to build images with BuildKit. There may be
situations where the Buildx component is not available, and BuildKit cannot be
used.

To provide a smooth transition to BuildKit as the default builder, Docker v23.0.0
has an automatic fallback for some situations, or produces an error to assist
users to resolve the problem.

In situations where the user did not explicitly opt-in to use BuildKit (i.e., 
`DOCKER_BUILDKIT=1` is not set), the CLI automatically falls back to the classic
builder, but prints a deprecation warning:

```
DEPRECATED: The legacy builder is deprecated and will be removed in a future release.
            Install the buildx component to build images with BuildKit:
            https://docs.docker.com/go/buildx/
```

This situation may occur if the `docker` CLI is installed using the static binaries,
and the Buildx component is not installed or not installed correctly. This fallback
will be removed in a future release, therefore we recommend to [install the Buildx component](https://docs.docker.com/go/buildx/)
and use BuildKit for your builds, or opt-out of using BuildKit with `DOCKER_BUILDKIT=0`.

If you opted-in to use BuildKit (`DOCKER_BUILDKIT=1`), but the Buildx component
is missing, an error is printed instead, and the `docker build` command fails:

```
ERROR: BuildKit is enabled but the buildx component is missing or broken.
       Install the buildx component to build images with BuildKit:
       https://docs.docker.com/go/buildx/
```

We recommend to [install the Buildx component](https://docs.docker.com/go/buildx/)
to continue using BuildKit for your builds, but alternatively, users can either
unset the `DOCKER_BUILDKIT` environment variable to fall back to the legacy builder,
or opt-out of using BuildKit with `DOCKER_BUILDKIT=0`.

Be aware that the [classic builder is deprecated](#legacy-builder-for-linux-images)
so both the automatic fallback and opting-out of using BuildKit will no longer
be possible in a future release.

### Btrfs storage driver on CentOS 7 and RHEL 7

**Removed in Release: v23.0.0**

The `btrfs` storage driver on CentOS and RHEL was provided as a technology preview
by CentOS and RHEL, but has been deprecated since the [Red Hat Enterprise Linux 7.4 release](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/7/html/storage_administration_guide/ch-btrfs),
and removed in CentOS 8 and RHEL 8. Users of the `btrfs` storage driver on CentOS
are recommended to migrate to a different storage driver, such as `overlay2`, which
is now the default storage driver. Docker 23.0 continues to provide the `btrfs`
storage driver to allow users to migrate to an alternative driver. The next release
of Docker will no longer provide this driver.

### Support for encrypted TLS private keys

**Deprecated in Release: v20.10**

**Removed in Release: v23.0.0**

Use of encrypted TLS private keys has been deprecated, and has been removed.
Golang has deprecated support for legacy PEM encryption (as specified in
[RFC 1423](https://datatracker.ietf.org/doc/html/rfc1423)), as it is insecure by
design (see [https://go-review.googlesource.com/c/go/+/264159](https://go-review.googlesource.com/c/go/+/264159)).

This feature allowed using an encrypted private key with a supplied password,
but did not provide additional security as the encryption is known to be broken,
and the key is sitting next to the password in the filesystem. Users are recommended
to decrypt the private key, and store it un-encrypted to continue using it.

### Kubernetes stack and context support

**Deprecated in Release: v20.10**
**Removed in Release: v23.0.0**

Following the deprecation of [Compose on Kubernetes](https://github.com/docker/compose-on-kubernetes),
support for Kubernetes in the `stack` and `context` commands has been removed from
the cli, and options related to this functionality are now either ignored, or may
produce an error.

The following command-line flags are removed from the `docker context` subcommands:

- `--default-stack-orchestrator` - swarm is now the only (and default) orchestrator for stacks.
- `--kubernetes` - the kubernetes endpoint can no longer be stored in `docker context`.
- `--kubeconfig` - exporting a context as a kubeconfig file is no longer supported.

The output produced by the `docker context inspect` subcommand no longer contains
information about `StackOrchestrator` and `Kubernetes` endpoints for new contexts.

The following command-line flags are removed from the `docker stack` subcommands:
 
- `--kubeconfig` - using a kubeconfig file as context is no longer supported.
- `--namespace` - configuring the kubernetes namespace for stacks is no longer supported.
- `--orchestrator` - swarm is now the only (and default) orchestrator for stacks.

The `DOCKER_STACK_ORCHESTRATOR`, `DOCKER_ORCHESTRATOR`, and `KUBECONFIG` environment
variables, as well as the `stackOrchestrator` option in the `~/.docker/config.json`
cli configuration file are no longer used, and ignored.

### Pulling images from non-compliant image registries

**Deprecated in Release: v20.10**

Docker Engine v20.10 and up includes optimizations to verify if images in the
local image cache need updating before pulling, preventing the Docker Engine
from making unnecessary API requests. These optimizations require the container
image registry to conform to the [Open Container Initiative Distribution Specification](https://github.com/opencontainers/distribution-spec).

While most registries conform to the specification, we encountered some registries
to be non-compliant, resulting in `docker pull` to fail.

As a temporary solution, Docker Engine v20.10 includes a fallback mechanism to
allow `docker pull` to be functional when using a non-compliant registry. A
warning message is printed in this situation:

    WARNING Failed to pull manifest by the resolved digest. This registry does not
            appear to conform to the distribution registry specification; falling back to
            pull by tag. This fallback is DEPRECATED, and will be removed in a future
            release.

The fallback is added to allow users to either migrate their images to a compliant
registry, or for these registries to become compliant.

Note that this fallback only addresses failures on `docker pull`. Other commands,
such as `docker stack deploy`, or pulling images with `containerd` will continue
to fail.

Given that other functionality is still broken with these registries, we consider
this fallback a _temporary_ solution, and will remove the fallback in an upcoming
major release.

### Linux containers on Windows (LCOW) (experimental)

**Deprecated in Release: v20.10**
**Removed in Release: v23.0.0**

The experimental feature to run Linux containers on Windows (LCOW) was introduced
as a technical preview in Docker 17.09. While many enhancements were made after
its introduction, the feature never reached completeness, and development has
now stopped in favor of running docker natively on Linux in WSL2.

Developers who want to run Linux workloads on a Windows host are encouraged to use
[Docker Desktop with WSL2](https://docs.docker.com/docker-for-windows/wsl/) instead.

### BLKIO weight options with cgroups v1

**Deprecated in Release: v20.10**

Specifying blkio weight (`docker run --blkio-weight` and `docker run --blkio-weight-device`)
is now marked as deprecated when using cgroups v1 because the corresponding features
were [removed in Linux kernel v5.0 and up](https://github.com/torvalds/linux/commit/f382fb0bcef4c37dc049e9f6963e3baf204d815c).
When using cgroups v2, the `--blkio-weight` options are implemented using
[`io.weight](https://github.com/torvalds/linux/blob/v5.0/Documentation/admin-guide/cgroup-v2.rst#io).

### Kernel memory limit

**Deprecated in Release: v20.10**
**Removed in Release: v23.0.0**

Specifying kernel memory limit (`docker run --kernel-memory`) is no longer supported
because the [Linux kernel deprecated `kmem.limit_in_bytes` in v5.4](https://github.com/torvalds/linux/commit/0158115f702b0ba208ab0b5adf44cae99b3ebcc7).
The OCI runtime specification now marks this option (as well as `--kernel-memory-tcp`)
as ["NOT RECOMMENDED"](https://github.com/opencontainers/runtime-spec/pull/1093),
and OCI runtimes such as `runc` no longer support this option.

Docker API v1.42 and up now ignores this option when set. Older versions of the
API continue to accept the option, but depending on the OCI runtime used, may
take no effect.

> **Note**
> 
> While not deprecated (yet) in Docker, the OCI runtime specification also
> deprecated the `memory.kmem.tcp.limit_in_bytes` option. When using `runc` as
> runtime, this option takes no effect. The linux kernel did not explicitly
> deprecate this feature, and there is a tracking ticket in the `runc` issue
> tracker to determine if this option should be reinstated or if this was an
> oversight of the Linux kernel maintainers (see [opencontainers/runc#3174](https://github.com/opencontainers/runc/issues/3174)).
> 
> The `memory.kmem.tcp.limit_in_bytes` option is only supported with cgroups v1,
> and not available on installations running with cgroups v2. This option is
> only supported by the API, and not exposed on the `docker` command-line.

### Classic Swarm and overlay networks using cluster store

**Deprecated in Release: v20.10**
**Removed in Release: v23.0.0**

Standalone ("classic") Swarm has been deprecated, and with that the use of overlay
networks using an external key/value store. The corresponding`--cluster-advertise`,
`--cluster-store`, and `--cluster-store-opt` daemon options have been removed.

### Support for legacy `~/.dockercfg` configuration files

**Deprecated in Release: v20.10**
**Removed in Release: v23.0.0**

The docker CLI up until v1.7.0 used the `~/.dockercfg` file to store credentials
after authenticating to a registry (`docker login`). Docker v1.7.0 replaced this
file with a new CLI configuration file, located in `~/.docker/config.json`. When
implementing the new configuration file, the old file (and file-format) was kept
as a fall-back, to assist existing users with migrating to the new file.

Given that the old file format encourages insecure storage of credentials
(credentials are stored unencrypted), and that no version of the CLI since
Docker v1.7.0 has created this file, support for this file, and its format has
been removed.

A warning is printed in situations where the CLI would fall back to the old file,
notifying the user that the legacy file is present, but ignored.

### Configuration options for experimental CLI features

**Deprecated in Release: v19.03**

**Removed in Release: v23.0.0**

The `DOCKER_CLI_EXPERIMENTAL` environment variable and the corresponding `experimental`
field in the CLI configuration file are deprecated. Experimental features are
enabled by default, and these configuration options are no longer functional.

Starting with v23.0.0, the Docker CLI no longer prints `Experimental` for the client
in the output of `docker version`, and the field has been removed from the JSON
format.

### CLI plugins support

**Deprecated in Release: v20.10**

CLI Plugin API is now marked as deprecated.

### Dockerfile legacy `ENV name value` syntax

**Deprecated in Release: v20.10**

The Dockerfile `ENV` instruction allows values to be set using either `ENV name=value`
or `ENV name value`. The latter (`ENV name value`) form can be ambiguous, for example,
the following defines a single env-variable (`ONE`) with value `"TWO= THREE=world"`,
but may have intended to be setting three env-vars:

```dockerfile
ENV ONE TWO= THREE=world
```

This format also does not allow setting multiple environment-variables in a single
`ENV` line in the Dockerfile.

Use of the `ENV name value` syntax is discouraged, and may be removed in a future
release. Users are encouraged to update their Dockerfiles to use the `ENV name=value`
syntax, for example:

```dockerfile
ENV ONE="" TWO="" THREE="world"
```

### `docker build --stream` flag (experimental)

**Deprecated in Release: v20.10**
**Removed in Release: v20.10**

Docker v17.07 introduced an experimental `--stream` flag on `docker build` which
allowed the build-context to be incrementally sent to the daemon, instead of
unconditionally sending the whole build-context.

This functionality has been reimplemented as part of BuildKit, which uses streaming
by default and the `--stream` option will be ignored when using the classic builder,
printing a deprecation warning instead.

Users that want to use this feature are encouraged to enable BuildKit by setting
the `DOCKER_BUILDKIT=1` environment variable or through the daemon or CLI configuration
files.

### `fluentd-async-connect` log opt

**Deprecated in Release: v20.10**

The `--log-opt fluentd-async-connect` option for the fluentd logging driver is
[deprecated in favor of `--log-opt fluentd-async`](https://github.com/moby/moby/pull/39086).
A deprecation message is logged in the daemon logs if the old option is used:

```console
fluent#New: AsyncConnect is now deprecated, please use Async instead
```

Users are encouraged to use the `fluentd-async` option going forward, as support
for the old option will be removed in a future release.

### Pushing and pulling with image manifest v2 schema 1

**Deprecated in Release: v19.03**

**Target For Removal In Release: v20.10**

The image manifest
[v2 schema 1](https://github.com/docker/distribution/blob/fda42e5ef908bdba722d435ff1f330d40dfcd56c/docs/spec/manifest-v2-1.md)
format is deprecated in favor of the
[v2 schema 2](https://github.com/docker/distribution/blob/fda42e5ef908bdba722d435ff1f330d40dfcd56c/docs/spec/manifest-v2-2.md) format.

If the registry you are using still supports v2 schema 1, urge their administrators to move to v2 schema 2.


### `docker engine` subcommands

**Deprecated in Release: v19.03**

**Removed in Release: v20.10**

The `docker engine activate`, `docker engine check`, and `docker engine update`
provided an alternative installation method to upgrade Docker Community engines
to Docker Enterprise, using an image-based distribution of the Docker Engine.

This feature was only available on Linux, and only when executed on a local node.
Given the limitations of this feature, and the feature not getting widely adopted,
the `docker engine` subcommands will be removed, in favor of installation through
standard package  managers.


### Top-level `docker deploy` subcommand (experimental)

**Deprecated in Release: v19.03**

**Removed in Release: v20.10**

The top-level `docker deploy` command (using the "Docker Application Bundle"
(.dab) file format was introduced as an experimental feature in Docker 1.13 /
17.03, but superseded by support for Docker Compose files using the `docker stack deploy`
subcommand.


### `docker stack deploy` using "dab" files (experimental)

**Deprecated in Release: v19.03**

**Removed in Release: v20.10**

With no development being done on this feature, and no active use of the file
format, support for the DAB file format and the top-level docker deploy command
(hidden by default in 19.03), will be removed, in favour of `docker stack deploy`
using compose files.

### Support for the `overlay2.override_kernel_check` storage option

**Deprecated in Release: v19.03**
**Disabled in Release: v19.03**

This daemon configuration option disabled the Linux kernel version check used
to detect if the kernel supported OverlayFS with multiple lower dirs, which is
required for the overlay2 storage driver. Starting with Docker v19.03.7, the
detection was improved to no longer depend on the kernel _version_, so this
option was no longer used.

Docker v23.0.0 logs a warning in the daemon logs if this option is set, and
users should remove this option from their daemon configuration.

### AuFS storage driver

**Deprecated in Release: v19.03**
**Disabled by default in Release: v23.0.0**

The `aufs` storage driver is deprecated in favor of `overlay2`, and will
be removed in a future release. Users of the `aufs` storage driver are
recommended to migrate to a different storage driver, such as `overlay2`, which
is now the default storage driver.

The `aufs` storage driver facilitates running Docker on distros that have no
support for OverlayFS, such as Ubuntu 14.04 LTS, which originally shipped with
a 3.14 kernel.

Now that Ubuntu 14.04 is no longer a supported distro for Docker, and `overlay2`
is available to all supported distros (as they are either on kernel 4.x, or have
support for multiple lowerdirs backported), there is no reason to continue
maintenance of the `aufs` storage driver.

#### Disabled by default in v23.0.0

Docker already prevented deprecated storage drivers from being automatically
selected on new installations, but continued to use these drivers when upgrading
existing installations. Starting with the v23.0.0 release, the Docker Engine will
fail to start if a deprecated storage driver is used (see [moby#43378](https://github.com/moby/moby/pull/43378):

```console
failed to start daemon: error initializing graphdriver: prior storage driver
aufs is deprecated and will be removed in a future release; update the the daemon
configuration and explicitly choose this storage driver to continue using it;
visit https://docs.docker.com/go/storage-driver/ for more information.
```

To continue using the storage driver, update the daemon configuration to use
explicitly use the given storage driver. Users are encouraged to migrate to 
different storage driver.

### Legacy "overlay" storage driver

**Deprecated in Release: v18.09**
**Disabled by default in Release: v23.0.0**

The `overlay` storage driver is deprecated in favor of the `overlay2` storage
driver, which has all the benefits of `overlay`, without its limitations (excessive
inode consumption). The legacy `overlay` storage driver will be removed in a future
release. Users of the `overlay` storage driver should migrate to the `overlay2`
storage driver.

The legacy `overlay` storage driver allowed using overlayFS-backed filesystems
on pre 4.x kernels. Now that all supported distributions are able to run `overlay2`
(as they are either on kernel 4.x, or have support for multiple lowerdirs
backported), there is no reason to keep maintaining the `overlay` storage driver.

#### Disabled by default in v23.0.0

Docker already prevented deprecated storage drivers from being automatically
selected on new installations, but continued to use these drivers when upgrading
existing installations. Starting with the v23.0.0 release, the Docker Engine will
fail to start if a deprecated storage driver is used (see [moby#43378](https://github.com/moby/moby/pull/43378):

```console
failed to start daemon: error initializing graphdriver: prior storage driver
overlay is deprecated and will be removed in a future release; update the the daemon
configuration and explicitly choose this storage driver to continue using it;
visit https://docs.docker.com/go/storage-driver/ for more information.
```

To continue using the storage driver, update the daemon configuration to use
explicitly use the given storage driver. Users are encouraged to migrate to
different storage driver.

### Device mapper storage driver

**Deprecated in Release: v18.09**
**Disabled by default in Release: v23.0.0**

The `devicemapper` storage driver is deprecated in favor of `overlay2`, and will
be removed in a future release. Users of the `devicemapper` storage driver are
recommended to migrate to a different storage driver, such as `overlay2`, which
is now the default storage driver.

The `devicemapper` storage driver facilitates running Docker on older (3.x) kernels
that have no support for other storage drivers (such as overlay2, or AUFS).

Now that support for `overlay2` is added to all supported distros (as they are
either on kernel 4.x, or have support for multiple lowerdirs backported), there
is no reason to continue maintenance of the `devicemapper` storage driver.

#### Disabled by default in v23.0.0

Docker already prevented deprecated storage drivers from being automatically
selected on new installations, but continued to use these drivers when upgrading
existing installations. Starting with the v23.0.0 release, the Docker Engine will
fail to start if a deprecated storage driver is used (see [moby#43378](https://github.com/moby/moby/pull/43378):

```console
failed to start daemon: error initializing graphdriver: prior storage driver
devicemapper is deprecated and will be removed in a future release; update the the daemon
configuration and explicitly choose this storage driver to continue using it;
visit https://docs.docker.com/go/storage-driver/ for more information.
```

To continue using the storage driver, update the daemon configuration to use
explicitly use the given storage driver. Users are encouraged to migrate to
different storage driver.

### Use of reserved namespaces in engine labels

**Deprecated in Release: v18.06**

**Removed In Release: v20.10**

The namespaces `com.docker.*`, `io.docker.*`, and `org.dockerproject.*` in engine labels
were always documented to be reserved, but there was never any enforcement.

Usage of these namespaces will now cause a warning in the engine logs to discourage their
use, and will error instead in v20.10 and above.


### `--disable-legacy-registry` override daemon option

**Disabled In Release: v17.12**

**Removed In Release: v19.03**

The `--disable-legacy-registry` flag was disabled in Docker 17.12 and will print
an error when used. For this error to be printed, the flag itself is still present,
but hidden. The flag has been removed in Docker 19.03.


### Interacting with V1 registries

**Disabled By Default In Release: v17.06**

**Removed In Release: v17.12**

Version 1.8.3 added a flag (`--disable-legacy-registry=false`) which prevents the
docker daemon from `pull`, `push`, and `login` operations against v1
registries.  Though enabled by default, this signals the intent to deprecate
the v1 protocol.

Support for the v1 protocol to the public registry was removed in 1.13. Any
mirror configurations using v1 should be updated to use a
[v2 registry mirror](https://docs.docker.com/registry/recipes/mirror/).

Starting with Docker 17.12, support for V1 registries has been removed, and the
`--disable-legacy-registry` flag can no longer be used, and `dockerd` will fail to
start when set.


### Asynchronous `service create` and `service update` as default

**Deprecated In Release: v17.05**

**Disabled by default in release: [v17.10](https://github.com/docker/docker-ce/releases/tag/v17.10.0-ce)**

Docker 17.05 added an optional `--detach=false` option to make the
`docker service create` and `docker service update` work synchronously. This
option will be enabled by default in Docker 17.10, at which point the `--detach`
flag can be used to use the previous (asynchronous) behavior.

The default for this option will also be changed accordingly for `docker service rollback`
and `docker service scale` in Docker 17.10.

### `-g` and `--graph` flags on `dockerd`

**Deprecated In Release: v17.05**

**Removed In Release: v23.0.0**

The `-g` or `--graph` flag for the `dockerd` or `docker daemon` command was
used to indicate the directory in which to store persistent data and resource
configuration and has been replaced with the more descriptive `--data-root`
flag. These flags were deprecated and hidden in v17.05, and removed in v23.0.0.

### Top-level network properties in NetworkSettings

**Deprecated In Release: [v1.13.0](https://github.com/docker/docker/releases/tag/v1.13.0)**

**Target For Removal In Release: v17.12**

When inspecting a container, `NetworkSettings` contains top-level information
about the default ("bridge") network;

`EndpointID`, `Gateway`, `GlobalIPv6Address`, `GlobalIPv6PrefixLen`, `IPAddress`,
`IPPrefixLen`, `IPv6Gateway`, and `MacAddress`.

These properties are deprecated in favor of per-network properties in
`NetworkSettings.Networks`. These properties were already "deprecated" in
docker 1.9, but kept around for backward compatibility.

Refer to [#17538](https://github.com/docker/docker/pull/17538) for further
information.

### `filter` param for `/images/json` endpoint
**Deprecated In Release: [v1.13.0](https://github.com/docker/docker/releases/tag/v1.13.0)**

**Removed In Release: v20.10**

The `filter` param to filter the list of image by reference (name or name:tag)
is now implemented as a regular filter, named `reference`.

### `repository:shortid` image references
**Deprecated In Release: [v1.13.0](https://github.com/docker/docker/releases/tag/v1.13.0)**

**Removed In Release: v17.12**

The `repository:shortid` syntax for referencing images is very little used,
collides with tag references, and can be confused with digest references.

Support for the `repository:shortid` notation to reference images was removed
in Docker 17.12.

### `docker daemon` subcommand
**Deprecated In Release: [v1.13.0](https://github.com/docker/docker/releases/tag/v1.13.0)**

**Removed In Release: v17.12**

The daemon is moved to a separate binary (`dockerd`), and should be used instead.

### Duplicate keys with conflicting values in engine labels
**Deprecated In Release: [v1.13.0](https://github.com/docker/docker/releases/tag/v1.13.0)**

**Removed In Release: v17.12**

When setting duplicate keys with conflicting values, an error will be produced, and the daemon
will fail to start.

### `MAINTAINER` in Dockerfile
**Deprecated In Release: [v1.13.0](https://github.com/docker/docker/releases/tag/v1.13.0)**

`MAINTAINER` was an early very limited form of `LABEL` which should be used instead.

### API calls without a version
**Deprecated In Release: [v1.13.0](https://github.com/docker/docker/releases/tag/v1.13.0)**

**Target For Removal In Release: v17.12**

API versions should be supplied to all API calls to ensure compatibility with
future Engine versions. Instead of just requesting, for example, the URL
`/containers/json`, you must now request `/v1.25/containers/json`.

### Backing filesystem without `d_type` support for overlay/overlay2
**Deprecated In Release: [v1.13.0](https://github.com/docker/docker/releases/tag/v1.13.0)**

**Removed In Release: v17.12**

The overlay and overlay2 storage driver does not work as expected if the backing
filesystem does not support `d_type`. For example, XFS does not support `d_type`
if it is formatted with the `ftype=0` option.

Support for these setups has been removed, and Docker v23.0.0 and up now fails to
start when attempting to use the `overlay2` or `overlay` storage driver on a
backing filesystem without `d_type` support.

Refer to [#27358](https://github.com/docker/docker/issues/27358) for details.


### `--automated` and `--stars` flags on `docker search`

**Deprecated in Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

**Removed In Release: v20.10**

The `docker search --automated` and `docker search --stars` options are deprecated.
Use `docker search --filter=is-automated=<true|false>` and `docker search --filter=stars=...` instead.


### `-h` shorthand for `--help`

**Deprecated In Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

**Target For Removal In Release: v17.09**

The shorthand (`-h`) is less common than `--help` on Linux and cannot be used
on all subcommands (due to it conflicting with, e.g. `-h` / `--hostname` on
`docker create`). For this reason, the `-h` shorthand was not printed in the
"usage" output of subcommands, nor documented, and is now marked "deprecated".

### `-e` and `--email` flags on `docker login`
**Deprecated In Release: [v1.11.0](https://github.com/docker/docker/releases/tag/v1.11.0)**

**Removed In Release: [v17.06](https://github.com/docker/docker-ce/releases/tag/v17.06.0-ce)**

The docker login command is removing the ability to automatically register for an account with the target registry if the given username doesn't exist. Due to this change, the email flag is no longer required, and will be deprecated.

### Separator (`:`) of `--security-opt` flag on `docker run`
**Deprecated In Release: [v1.11.0](https://github.com/docker/docker/releases/tag/v1.11.0)**

**Target For Removal In Release: v17.06**

The flag `--security-opt` doesn't use the colon separator (`:`) anymore to divide keys and values, it uses the equal symbol (`=`) for consistency with other similar flags, like `--storage-opt`.

### Ambiguous event fields in API
**Deprecated In Release: [v1.10.0](https://github.com/docker/docker/releases/tag/v1.10.0)**

The fields `ID`, `Status` and `From` in the events API have been deprecated in favor of a more rich structure.
See the events API documentation for the new format.

### `-f` flag on `docker tag`
**Deprecated In Release: [v1.10.0](https://github.com/docker/docker/releases/tag/v1.10.0)**

**Removed In Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

To make tagging consistent across the various `docker` commands, the `-f` flag on the `docker tag` command is deprecated. It is no longer necessary to specify `-f` to move a tag from one image to another. Nor will `docker` generate an error if the `-f` flag is missing and the specified tag is already in use.

### HostConfig at API container start
**Deprecated In Release: [v1.10.0](https://github.com/docker/docker/releases/tag/v1.10.0)**

**Removed In Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

Passing an `HostConfig` to `POST /containers/{name}/start` is deprecated in favor of
defining it at container creation (`POST /containers/create`).

### `--before` and `--since` flags on `docker ps`

**Deprecated In Release: [v1.10.0](https://github.com/docker/docker/releases/tag/v1.10.0)**

**Removed In Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

The `docker ps --before` and `docker ps --since` options are deprecated.
Use `docker ps --filter=before=...` and `docker ps --filter=since=...` instead.


### Driver-specific log tags
**Deprecated In Release: [v1.9.0](https://github.com/docker/docker/releases/tag/v1.9.0)**

**Removed In Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

Log tags are now generated in a standard way across different logging drivers.
Because of which, the driver specific log tag options `syslog-tag`, `gelf-tag` and
`fluentd-tag` have been deprecated in favor of the generic `tag` option.

```console
{% raw %}
$ docker --log-driver=syslog --log-opt tag="{{.ImageName}}/{{.Name}}/{{.ID}}"
{% endraw %}
```


### Docker Content Trust ENV passphrase variables name change
**Deprecated In Release: [v1.9.0](https://github.com/docker/docker/releases/tag/v1.9.0)**

**Removed In Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

Since 1.9, Docker Content Trust Offline key has been renamed to Root key and the Tagging key has been renamed to Repository key. Due to this renaming, we're also changing the corresponding environment variables

- DOCKER_CONTENT_TRUST_OFFLINE_PASSPHRASE is now named DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE
- DOCKER_CONTENT_TRUST_TAGGING_PASSPHRASE is now named DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE


### `/containers/(id or name)/copy` endpoint

**Deprecated In Release: [v1.8.0](https://github.com/docker/docker/releases/tag/v1.8.0)**

**Removed In Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

The endpoint `/containers/(id or name)/copy` is deprecated in favor of `/containers/(id or name)/archive`.


### LXC built-in exec driver
**Deprecated In Release: [v1.8.0](https://github.com/docker/docker/releases/tag/v1.8.0)**

**Removed In Release: [v1.10.0](https://github.com/docker/docker/releases/tag/v1.10.0)**

The built-in LXC execution driver, the lxc-conf flag, and API fields have been removed.


### Old Command Line Options
**Deprecated In Release: [v1.8.0](https://github.com/docker/docker/releases/tag/v1.8.0)**

**Removed In Release: [v1.10.0](https://github.com/docker/docker/releases/tag/v1.10.0)**

The flags `-d` and `--daemon` are deprecated in favor of the `daemon` subcommand:

    docker daemon -H ...

The following single-dash (`-opt`) variant of certain command line options
are deprecated and replaced with double-dash options (`--opt`):

    docker attach -nostdin
    docker attach -sig-proxy
    docker build -no-cache
    docker build -rm
    docker commit -author
    docker commit -run
    docker events -since
    docker history -notrunc
    docker images -notrunc
    docker inspect -format
    docker ps -beforeId
    docker ps -notrunc
    docker ps -sinceId
    docker rm -link
    docker run -cidfile
    docker run -dns
    docker run -entrypoint
    docker run -expose
    docker run -link
    docker run -lxc-conf
    docker run -n
    docker run -privileged
    docker run -volumes-from
    docker search -notrunc
    docker search -stars
    docker search -t
    docker search -trusted
    docker tag -force

The following double-dash options are deprecated and have no replacement:

    docker run --cpuset
    docker run --networking
    docker ps --since-id
    docker ps --before-id
    docker search --trusted

**Deprecated In Release: [v1.5.0](https://github.com/docker/docker/releases/tag/v1.5.0)**

**Removed In Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

The single-dash (`-help`) was removed, in favor of the double-dash `--help`

    docker -help
    docker [COMMAND] -help


### `--api-enable-cors` flag on dockerd

**Deprecated In Release: [v1.6.0](https://github.com/docker/docker/releases/tag/v1.6.0)**

**Removed In Release: [v17.09](https://github.com/docker/docker-ce/releases/tag/v17.09.0-ce)**

The flag `--api-enable-cors` is deprecated since v1.6.0. Use the flag
`--api-cors-header` instead.

### `--run` flag on docker commit

**Deprecated In Release: [v0.10.0](https://github.com/docker/docker/releases/tag/v0.10.0)**

**Removed In Release: [v1.13.0](https://github.com/docker/docker/releases/tag/v1.13.0)**

The flag `--run` of the docker commit (and its short version `-run`) were deprecated in favor
of the `--changes` flag that allows to pass `Dockerfile` commands.


### Three arguments form in `docker import`
**Deprecated In Release: [v0.6.7](https://github.com/docker/docker/releases/tag/v0.6.7)**

**Removed In Release: [v1.12.0](https://github.com/docker/docker/releases/tag/v1.12.0)**

The `docker import` command format `file|URL|- [REPOSITORY [TAG]]` is deprecated since November 2013. It's no more supported.
