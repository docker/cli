# Docker CLI

[![PkgGoDev](https://pkg.go.dev/badge/github.com/docker/cli)](https://pkg.go.dev/github.com/docker/cli)
[![Build Status](https://img.shields.io/github/actions/workflow/status/docker/cli/build.yml?branch=master&label=build&logo=github)](https://github.com/docker/cli/actions?query=workflow%3Abuild)
[![Test Status](https://img.shields.io/github/actions/workflow/status/docker/cli/test.yml?branch=master&label=test&logo=github)](https://github.com/docker/cli/actions?query=workflow%3Atest)
[![Go Report Card](https://goreportcard.com/badge/github.com/docker/cli)](https://goreportcard.com/report/github.com/docker/cli)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/docker/cli/badge)](https://scorecard.dev/viewer/?uri=github.com/docker/cli)
[![Codecov](https://img.shields.io/codecov/c/github/docker/cli?logo=codecov)](https://codecov.io/gh/docker/cli)

## About

This repository is the home of the Docker CLI.

## Development

`docker/cli` is developed using Docker.

Build CLI from source:

```shell
docker buildx bake
```

Build binaries for all supported platforms:

```shell
docker buildx bake cross
```

Build for a specific platform:

```shell
docker buildx bake --set binary.platform=linux/arm64 
```

Build dynamic binary for glibc or musl:

```shell
USE_GLIBC=1 docker buildx bake dynbinary 
```

Run all linting:

```shell
docker buildx bake lint shellcheck
```

Run test:

```shell
docker buildx bake test
```

List all the available targets:

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

## Licensing

docker/cli is licensed under the Apache License, Version 2.0. See
[LICENSE](https://github.com/docker/docker/blob/master/LICENSE) for the full
license text.
