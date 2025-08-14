// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

package command

import (
	"github.com/docker/cli/cli/command/internal/cli"
)

type (
	Streams    = cli.Streams
	Cli        = cli.Cli
	CLIOption  = cli.CLIOption
	ServerInfo = cli.ServerInfo
)

// deprecated: use [NewDockerCli] instead
type DockerCli = cli.DockerCli

var NewDockerCli = cli.NewDockerCli

var NewAPIClientFromFlags = cli.NewAPIClientFromFlags

// deprecated: use [cobra.Command.SetOut] and [cobra.Command.HelpFunc] instead
//
// example:
//
//	cmd.SetOut(os.Stderr)
//	cmd.HelpFunc()(cmd, args)
var ShowHelp = cli.ShowHelp

var WithGlobalMeterProvider = cli.WithGlobalMeterProvider
