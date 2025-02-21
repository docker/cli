package stack

import (
	"fmt"

	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/docker/cli/v28/cli/command/completion"
	"github.com/docker/cli/v28/cli/command/stack/swarm"
	"github.com/spf13/cobra"
)

// NewStackCommand returns a cobra command for `stack` subcommands
func NewStackCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack [OPTIONS]",
		Short: "Manage Swarm stacks",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version": "1.25",
			"swarm":   "manager",
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
		newConfigCommand(dockerCli),
	)
	flags := cmd.PersistentFlags()
	flags.String("orchestrator", "", "Orchestrator to use (swarm|all)")
	flags.SetAnnotation("orchestrator", "deprecated", nil)
	flags.MarkDeprecated("orchestrator", "option will be ignored")
	return cmd
}

// completeNames offers completion for swarm stacks
func completeNames(dockerCLI completion.APIClientProvider) completion.ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := swarm.GetStacks(cmd.Context(), dockerCLI.Client())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, stack := range list {
			names = append(names, stack.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
