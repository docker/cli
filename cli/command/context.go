// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

package command

import "github.com/docker/cli/cli/command/internal/cli"

type DockerContext = cli.DockerContext

var GetDockerContext = cli.GetDockerContext
