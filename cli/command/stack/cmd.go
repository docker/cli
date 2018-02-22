package stack

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewStackCommand returns a cobra command for `stack` subcommands
func NewStackCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Manage Docker stacks",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"kubernetes": "",
			"swarm":      "",
			"version":    "1.25",
		},
	}
	cmd.AddCommand(
		newDeployCommand(dockerCli),
		newListCommand(dockerCli),
		newPsCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newServicesCommand(dockerCli),
	)
	flags := cmd.PersistentFlags()
	flags.String("namespace", "default", "Kubernetes namespace to use")
	flags.SetAnnotation("namespace", "kubernetes", nil)
	flags.SetAnnotation("namespace", "experimentalCLI", nil)
	flags.String("kubeconfig", "", "Kubernetes config file")
	flags.SetAnnotation("kubeconfig", "kubernetes", nil)
	flags.SetAnnotation("kubeconfig", "experimentalCLI", nil)
	return cmd
}

// NewTopLevelDeployCommand returns a command for `docker deploy`
func NewTopLevelDeployCommand(dockerCli command.Cli) *cobra.Command {
	cmd := newDeployCommand(dockerCli)
	// Remove the aliases at the top level
	cmd.Aliases = []string{}
	cmd.Annotations = map[string]string{
		"experimental": "",
		"swarm":        "",
		"version":      "1.25",
	}
	return cmd
}
