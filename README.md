[![build status](https://circleci.com/gh/docker/cli.svg?style=shield)](https://circleci.com/gh/docker/cli/tree/master)

docker/cli
==========

This repository is the home of the cli used in the Docker CE and
Docker EE products.

Development
===========

The `./tasks` script allows you to build and develop the cli with Docker.

Build a linux binary:
```
$ ./tasks binary
```

Run all linting:
```
$ ./tasks lint
```

You can see a full list of tasks with `./tasks --help`.

### In-container development environment

Start an interactive development environment:

```
$ ./tasks shell
```

From the interactive development shell you can run tasks defined in the
Makefile. For example, to build a binary you would run:

```
$ make binary
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
