# Docker CLI

[![PkgGoDev](https://pkg.go.dev/badge/github.com/docker/cli)](https://pkg.go.dev/github.com/docker/cli)
[![Build Status](https://img.shields.io/github/actions/workflow/status/docker/cli/build.yml?branch=master&label=build&logo=github)](https://github.com/docker/cli/actions?query=workflow%3Abuild)
[![Test Status](https://img.shields.io/github/actions/workflow/status/docker/cli/test.yml?branch=master&label=test&logo=github)](https://github.com/docker/cli/actions?query=workflow%3Atest)
[![Go Report Card](https://goreportcard.com/badge/github.com/docker/cli)](https://goreportcard.com/report/github.com/docker/cli)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/docker/cli/badge)](https://scorecard.dev/viewer/?uri=github.com/docker/cli)
[![Codecov](https://img.shields.io/codecov/c/github/docker/cli?logo=codecov)](https://codecov.io/gh/docker/cli)

<img width="396" height="120" alt="docker-logo-blue" src="https://github.com/user-attachments/assets/90d7cb71-7613-4b93-9ab9-d635e00d2f23" />

---

# 📌 Table of Contents

- [About](#about)
- [Features](#features)
- [Development](#development)
  - [Build CLI from Source](#build-cli-from-source)
  - [Build Binaries for All supported Platforms](#build-binaries-for-all-supported-platforms)
  - [Build for a Specific Platform](#build-for-a-specific-platform)
  - [Build Dynamic Binary for glibc or musl](#build-dynamic-binary-for-glibc-or-musl)
  - [Run all Linting](#run-all-linting)
  - [Run Test](#run-test)
  - [List all the Available Targets](#list-all-the-available-targets)
  - [In-container Development Environment](#in-container-development-environment)
- [Legal](#legal)
- [Community](#community)
- [Licensing](#licensing)


---

## About

The Docker CLI is the official command-line interface used to interact with the Docker Engine. It provides developers, DevOps engineers, and system administrators with a powerful tool to manage containerized applications and resources such as images, containers, networks, and volumes.

This CLI is a key component of the Docker platform and is widely used in development environments, CI/CD pipelines, and production systems across various operating systems and architectures.

This repository contains the source code and tooling required to build, test, and contribute to the Docker CLI. It supports cross-compilation, custom builds, and offers a developer-friendly workflow using Docker itself.

## Features

Some of the key features of the Docker CLI include:

-  **Container Lifecycle Management**  
  Create, start, stop, restart, and remove containers with simple commands.

-  **Image Management**  
  Build, pull, push, and inspect Docker images across local and remote registries.

-  **Multi-Platform Builds**  
  Cross-compile and build images for multiple platforms using `buildx`.

-  **Network and Volume Control**  
  Manage container networking, custom bridge networks, and persistent storage volumes.

-  **Inspect and Debug**  
  Use commands like `docker inspect`, `logs`, and `exec` to investigate running containers.

-  **Scriptable and CI/CD Friendly**  
  Seamless integration with CI/CD tools for automated testing, building, and deployment.

-  **Extensive Documentation and Plugin Support**  
  Well-documented CLI commands and support for extensions and custom plugins.

-  **Docker Contexts**  
  Easily switch between different Docker endpoints (e.g., local, remote, cloud).

-  **Secure by Design**  
  Actively maintained with security in mind and part of the OpenSSF Scorecard program.


## Development

`docker/cli` is developed using Docker.

### Build CLI from source:

```shell
docker buildx bake
```

### Build binaries for all supported platforms:

```shell
docker buildx bake cross
```

### Build for a specific platform:

```shell
docker buildx bake --set binary.platform=linux/arm64 
```

### Build dynamic binary for glibc or musl:

```shell
USE_GLIBC=1 docker buildx bake dynbinary 
```

### Run all linting:

```shell
docker buildx bake lint shellcheck
```

### Run test:

```shell
docker buildx bake test
```

### List all the available targets:

```shell
make help
```

### In-container development environment

Start an interactive development environment:

```shell
make -f docker.Makefile shell
```

## Legal

*Brought to you courtesy of our legal counsel. For more context,
see the [NOTICE](https://github.com/docker/cli/blob/master/NOTICE) document in this repo.*

Use and transfer of Docker may be subject to certain restrictions by the
United States and other governments.

It is your responsibility to ensure that your use and/or transfer does not
violate applicable laws.

For more information, see https://www.bis.doc.gov

## Community

We welcome contributions, feedback, and collaboration from the community!

### Where to connect

- [GitHub Discussions](https://github.com/docker/cli/discussions) – Ask questions, share ideas, or get help.
- [Contributing Guide](https://github.com/docker/cli/blob/master/CONTRIBUTING.md) – Learn how to contribute to the Docker CLI.
- [Docker Community Forums](https://forums.docker.com) – Join conversations with the broader Docker community.
- [Docker Community Slack](https://dockr.ly/slack) – Chat with other developers in real time.
- [Issues](https://github.com/docker/cli/issues) – Found a bug? Report it here.

We’re excited to see what you’ll build and contribute with Docker CLI!

## Licensing

docker/cli is licensed under the Apache License, Version 2.0. See
[LICENSE](https://github.com/docker/docker/blob/master/LICENSE) for the full
license text.
