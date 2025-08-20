package commands

import (
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/builder"
	"github.com/docker/cli/cli/command/checkpoint"
	"github.com/docker/cli/cli/command/config"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/context"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/command/manifest"
	"github.com/docker/cli/cli/command/network"
	"github.com/docker/cli/cli/command/node"
	"github.com/docker/cli/cli/command/plugin"
	"github.com/docker/cli/cli/command/registry"
	"github.com/docker/cli/cli/command/secret"
	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/swarm"
	"github.com/docker/cli/cli/command/system"
	"github.com/docker/cli/cli/command/trust"
	"github.com/docker/cli/cli/command/volume"
	"github.com/spf13/cobra"
)

// AddCommands adds all the commands from cli/command to the root command
func AddCommands(cmd *cobra.Command, dockerCli command.Cli) {
	cmd.AddCommand(
		// commonly used shorthands
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		container.NewRunCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		container.NewExecCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		container.NewPsCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		image.NewBuildCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		image.NewPullCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		image.NewPushCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		image.NewImagesCommand(dockerCli),
		registry.NewLoginCommand(dockerCli),
		registry.NewLogoutCommand(dockerCli),
		registry.NewSearchCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		system.NewVersionCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		system.NewInfoCommand(dockerCli),

		// management commands
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		builder.NewBakeStubCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		builder.NewBuilderCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		checkpoint.NewCheckpointCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		container.NewContainerCommand(dockerCli),
		context.NewContextCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		image.NewImageCommand(dockerCli),
		manifest.NewManifestCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		network.NewNetworkCommand(dockerCli),
		plugin.NewPluginCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		system.NewSystemCommand(dockerCli),
		trust.NewTrustCommand(dockerCli),
		volume.NewVolumeCommand(dockerCli),

		// orchestration (swarm) commands
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		config.NewConfigCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		node.NewNodeCommand(dockerCli),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		secret.NewSecretCommand(dockerCli),
		service.NewServiceCommand(dockerCli),
		stack.NewStackCommand(dockerCli),
		swarm.NewSwarmCommand(dockerCli),

		// legacy commands may be hidden
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewAttachCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewCommitCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewCopyCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewCreateCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewDiffCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewExportCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewKillCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewLogsCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewPauseCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewPortCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewRenameCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewRestartCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewRmCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewStartCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewStatsCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewStopCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewTopCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewUnpauseCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewUpdateCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(container.NewWaitCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(image.NewHistoryCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(image.NewImportCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(image.NewLoadCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(image.NewRemoveCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(image.NewSaveCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(image.NewTagCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(system.NewEventsCommand(dockerCli)),
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		hide(system.NewInspectCommand(dockerCli)),
	)
}

func hide(cmd *cobra.Command) *cobra.Command {
	// If the environment variable with name "DOCKER_HIDE_LEGACY_COMMANDS" is not empty,
	// these legacy commands (such as `docker ps`, `docker exec`, etc)
	// will not be shown in output console.
	if os.Getenv("DOCKER_HIDE_LEGACY_COMMANDS") == "" {
		return cmd
	}
	cmdCopy := *cmd
	cmdCopy.Hidden = true
	cmdCopy.Aliases = []string{}
	return &cmdCopy
}
