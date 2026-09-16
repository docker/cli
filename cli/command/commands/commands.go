package commands

import (
	"github.com/docker/cli/v29/cli/command"
	_ "github.com/docker/cli/v29/cli/command/builder"
	_ "github.com/docker/cli/v29/cli/command/checkpoint"
	_ "github.com/docker/cli/v29/cli/command/config"
	_ "github.com/docker/cli/v29/cli/command/container"
	_ "github.com/docker/cli/v29/cli/command/context"
	_ "github.com/docker/cli/v29/cli/command/image"
	_ "github.com/docker/cli/v29/cli/command/manifest"
	_ "github.com/docker/cli/v29/cli/command/network"
	_ "github.com/docker/cli/v29/cli/command/node"
	_ "github.com/docker/cli/v29/cli/command/plugin"
	_ "github.com/docker/cli/v29/cli/command/registry"
	_ "github.com/docker/cli/v29/cli/command/secret"
	_ "github.com/docker/cli/v29/cli/command/service"
	_ "github.com/docker/cli/v29/cli/command/stack"
	_ "github.com/docker/cli/v29/cli/command/swarm"
	_ "github.com/docker/cli/v29/cli/command/system"
	_ "github.com/docker/cli/v29/cli/command/volume"
	"github.com/docker/cli/v29/internal/commands"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, dockerCLI command.Cli) {
	for _, c := range commands.Commands() {
		cmd.AddCommand(c(dockerCLI))
	}
}
