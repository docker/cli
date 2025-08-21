package stack

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/docker/cli/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newStackCommand)
}

// newStackCommand returns a cobra command for `stack` subcommands
func newStackCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack [OPTIONS]",
		Short: "Manage Swarm stacks",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{
			"version": "1.25",
			"swarm":   "manager",
		},
	}
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if err := cmd.Root().PersistentPreRunE(c, args); err != nil {
			fmt.Fprintln(dockerCLI.Err(), err)
			return
		}
		defaultHelpFunc(c, args)
	})
	cmd.AddCommand(
		newDeployCommand(dockerCLI),
		newListCommand(dockerCLI),
		newPsCommand(dockerCLI),
		newRemoveCommand(dockerCLI),
		newServicesCommand(dockerCLI),
		newConfigCommand(dockerCLI),
	)
	flags := cmd.PersistentFlags()
	flags.String("orchestrator", "", "Orchestrator to use (swarm|all)")
	flags.SetAnnotation("orchestrator", "deprecated", nil)
	flags.MarkDeprecated("orchestrator", "option will be ignored")
	return cmd
}

// completeNames offers completion for swarm stacks
func completeNames(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
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
