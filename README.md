[![build status](https://circleci.com/gh/docker/cli.svg?style=shield)](https://circleci.com/gh/docker/cli/tree/master)
[![Build Status](https://ci.docker.com/public/job/cli/job/master/badge/icon)](https://ci.docker.com/public/job/cli/job/master)

docker/cli
==========

This repository is the home of the cli used in the Docker CE and
Docker EE products.

Development
===========

`docker/cli` is developed using Docker.

Build CLI from source:

```
$ docker buildx bake
```

Build binaries for all supported platforms:

```
$ docker buildx bake cross
```

Build for a specific platform:

```
$ docker buildx bake --set binary.platform=linux/arm64 
```

Build dynamic binary for glibc or musl:

```
$ USE_GLIBC=1 docker buildx bake dynbinary 
```


Run all linting:

```
$ make -f docker.Makefile lint
```

List all the available targets:

```
$ make help
```

### In-container development environment

Start an interactive development environment:

```
$ make -f docker.Makefile shell
```

Legal
=====
*Brought to you courtesy of our legal counsel. For more context,
please see the [NOTICE](https://github.com/docker/cli/blob/master/NOTICE) document in this repo.*

Use and transfer of Docker may be subject to certain restrictions by the
United States and other governments.

It is your responsibility to ensure that your use and/or transfer does not
violate applicable laws.

For more information, please see https://www.bis.doc.gov

Licensing
=========
docker/cli is licensed under the Apache License, Version 2.0. See
[LICENSE](https://github.com/docker/docker/blob/master/LICENSE) for the full
license text.
