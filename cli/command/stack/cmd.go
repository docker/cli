package stack

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewStackCommand returns a cobra command for `stack` subcommands
func NewStackCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack [OPTIONS]",
		Short: "Manage Docker stacks",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version": "1.25",
		},
	}
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if err := cmd.Root().PersistentPreRunE(c, args); err != nil {
			fmt.Fprintln(dockerCli.Err(), err)
			return
		}
		defaultHelpFunc(c, args)
	})
	cmd.AddCommand(
		newDeployCommand(dockerCli),
		newListCommand(dockerCli),
		newPsCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newServicesCommand(dockerCli),
	)
	flags := cmd.PersistentFlags()
	flags.String("orchestrator", "", "Orchestrator to use (swarm|all)")
	flags.SetAnnotation("orchestrator", "deprecated", nil)
	flags.MarkDeprecated("orchestrator", "option will be ignored")
	return cmd
}
