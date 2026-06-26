[![PkgGoDev](https://img.shields.io/badge/go.dev-docs-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/docker/cli-docs-tool)
[![Test Status](https://img.shields.io/github/actions/workflow/status/docker/cli-docs-tool/test.yml?branch=main&label=test&logo=github&style=flat-square)](https://github.com/docker/cli-docs-tool/actions?query=workflow%3Atest)
[![Go Report Card](https://goreportcard.com/badge/github.com/docker/cli-docs-tool)](https://goreportcard.com/report/github.com/docker/cli-docs-tool)

## About

This is a library containing utilities to generate (reference) documentation
for the [`docker` CLI](https://github.com/docker/cli) on [docs.docker.com](https://docs.docker.com/reference/).

## Disclaimer

This library is intended for use by Docker's CLIs, and is not intended to be a
general-purpose utility. Various bits are hard-coded or make assumptions that
are very specific to our use-case. Contributions are welcome, but we will not
accept contributions to make this a general-purpose module.

## Usage

To generate the documentation it's recommended to do so using a Go submodule
in your repository.

We will use the example of `docker/buildx` and create a Go submodule in a
`docs` folder (recommended):

```console
$ mkdir docs
$ cd ./docs
$ go mod init github.com/docker/buildx/docs
$ go get github.com/docker/cli-docs-tool
```

Your `go.mod` should look like this:

```text
module github.com/docker/buildx/docs

go 1.16

require (
	github.com/docker/cli-docs-tool v0.0.0
)
```

Next, create a file named `main.go` inside that directory containing the
following Go code from [`example/main.go`](example/main.go).

Running this example should produce the following output:

```console
$ go run main.go
INFO: Generating Markdown for "docker buildx bake"
INFO: Generating Markdown for "docker buildx build"
INFO: Generating Markdown for "docker buildx create"
INFO: Generating Markdown for "docker buildx du"
...
INFO: Generating YAML for "docker buildx uninstall"
INFO: Generating YAML for "docker buildx use"
INFO: Generating YAML for "docker buildx version"
INFO: Generating YAML for "docker buildx"
```

Generated docs will be available in the `./docs` folder of the project.

## Contributing

Want to contribute? Awesome! You can find information about contributing to
this project in the [CONTRIBUTING.md](/.github/CONTRIBUTING.md)
