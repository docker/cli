package commands

import (
	"github.com/docker/cli/cli/command"
	_ "github.com/docker/cli/cli/command/builder"
	_ "github.com/docker/cli/cli/command/checkpoint"
	_ "github.com/docker/cli/cli/command/config"
	_ "github.com/docker/cli/cli/command/container"
	_ "github.com/docker/cli/cli/command/context"
	_ "github.com/docker/cli/cli/command/image"
	_ "github.com/docker/cli/cli/command/manifest"
	_ "github.com/docker/cli/cli/command/network"
	_ "github.com/docker/cli/cli/command/node"
	_ "github.com/docker/cli/cli/command/plugin"
	_ "github.com/docker/cli/cli/command/registry"
	_ "github.com/docker/cli/cli/command/secret"
	_ "github.com/docker/cli/cli/command/service"
	_ "github.com/docker/cli/cli/command/stack"
	_ "github.com/docker/cli/cli/command/swarm"
	_ "github.com/docker/cli/cli/command/system"
	_ "github.com/docker/cli/cli/command/trust"
	_ "github.com/docker/cli/cli/command/volume"
	"github.com/docker/cli/internal/commands"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, dockerCLI command.Cli) {
	for _, c := range commands.Commands() {
		cmd.AddCommand(c(dockerCLI))
	}
}
