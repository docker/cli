package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/orchestrator"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/spf13/cobra"
)

// NewStackCommand returns a cobra command for `stack` subcommands
func NewStackCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "stack",
		Short:       "Manage Docker stacks",
		Args:        cli.NoArgs,
		RunE:        command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{"version": "1.25"},
	}
	switch orchestrator.GetOrchestrator(dockerCli) {
	case orchestrator.Kubernetes:
		kubernetes.AddStackCommands(cmd, dockerCli)
	case orchestrator.Swarm:
		swarm.AddStackCommands(cmd, dockerCli)
	}
	return cmd
}

// NewTopLevelDeployCommand returns a command for `docker deploy`
func NewTopLevelDeployCommand(dockerCli command.Cli) *cobra.Command {
	var cmd *cobra.Command
	switch orchestrator.GetOrchestrator(dockerCli) {
	case orchestrator.Kubernetes:
		cmd = kubernetes.NewTopLevelDeployCommand(dockerCli)
	case orchestrator.Swarm:
		cmd = swarm.NewTopLevelDeployCommand(dockerCli)
	}
	// Remove the aliases at the top level
	cmd.Aliases = []string{}
	cmd.Annotations = map[string]string{"experimental": "", "version": "1.25"}
	return cmd
}
