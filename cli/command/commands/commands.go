package commands

import (
	"os"
	"runtime"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/builder"
	"github.com/docker/cli/cli/command/checkpoint"
	"github.com/docker/cli/cli/command/config"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/context"
	"github.com/docker/cli/cli/command/engine"
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
		// checkpoint
		checkpoint.NewCheckpointCommand(dockerCli),

		// config
		config.NewConfigCommand(dockerCli),

		// container
		container.NewContainerCommand(dockerCli),
		container.NewRunCommand(dockerCli),

		// image
		image.NewImageCommand(dockerCli),
		image.NewBuildCommand(dockerCli),

		// builder
		builder.NewBuilderCommand(dockerCli),

		// manifest
		manifest.NewManifestCommand(dockerCli),

		// network
		network.NewNetworkCommand(dockerCli),

		// node
		node.NewNodeCommand(dockerCli),

		// plugin
		plugin.NewPluginCommand(dockerCli),

		// registry
		registry.NewLoginCommand(dockerCli),
		registry.NewLogoutCommand(dockerCli),
		registry.NewSearchCommand(dockerCli),

		// secret
		secret.NewSecretCommand(dockerCli),

		// service
		service.NewServiceCommand(dockerCli),

		// system
		system.NewSystemCommand(dockerCli),
		system.NewVersionCommand(dockerCli),

		// stack
		stack.NewStackCommand(dockerCli),

		// swarm
		swarm.NewSwarmCommand(dockerCli),

		// trust
		trust.NewTrustCommand(dockerCli),

		// volume
		volume.NewVolumeCommand(dockerCli),

		// context
		context.NewContextCommand(dockerCli),

		// legacy commands may be hidden
		hide(stack.NewTopLevelDeployCommand(dockerCli)),
		hide(system.NewEventsCommand(dockerCli)),
		hide(system.NewInfoCommand(dockerCli)),
		hide(system.NewInspectCommand(dockerCli)),
		hide(container.NewAttachCommand(dockerCli)),
		hide(container.NewCommitCommand(dockerCli)),
		hide(container.NewCopyCommand(dockerCli)),
		hide(container.NewCreateCommand(dockerCli)),
		hide(container.NewDiffCommand(dockerCli)),
		hide(container.NewExecCommand(dockerCli)),
		hide(container.NewExportCommand(dockerCli)),
		hide(container.NewKillCommand(dockerCli)),
		hide(container.NewLogsCommand(dockerCli)),
		hide(container.NewPauseCommand(dockerCli)),
		hide(container.NewPortCommand(dockerCli)),
		hide(container.NewPsCommand(dockerCli)),
		hide(container.NewRenameCommand(dockerCli)),
		hide(container.NewRestartCommand(dockerCli)),
		hide(container.NewRmCommand(dockerCli)),
		hide(container.NewStartCommand(dockerCli)),
		hide(container.NewStatsCommand(dockerCli)),
		hide(container.NewStopCommand(dockerCli)),
		hide(container.NewTopCommand(dockerCli)),
		hide(container.NewUnpauseCommand(dockerCli)),
		hide(container.NewUpdateCommand(dockerCli)),
		hide(container.NewWaitCommand(dockerCli)),
		hide(image.NewHistoryCommand(dockerCli)),
		hide(image.NewImagesCommand(dockerCli)),
		hide(image.NewImportCommand(dockerCli)),
		hide(image.NewLoadCommand(dockerCli)),
		hide(image.NewPullCommand(dockerCli)),
		hide(image.NewPushCommand(dockerCli)),
		hide(image.NewRemoveCommand(dockerCli)),
		hide(image.NewSaveCommand(dockerCli)),
		hide(image.NewTagCommand(dockerCli)),
	)
	if runtime.GOOS == "linux" {
		// engine
		cmd.AddCommand(engine.NewEngineCommand(dockerCli))
	}
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
