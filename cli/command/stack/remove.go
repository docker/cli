package stack

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/legacy/kubernetes"
	"github.com/docker/cli/cli/command/stack/legacy/swarm"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/stacks/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newRemoveCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
	var opts options.Remove

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] STACK [STACK...]",
		Aliases: []string{"remove", "down"},
		Short:   "Remove one or more stacks",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespaces = args
			if err := validateStackNames(opts.Namespaces); err != nil {
				return err
			}
			return RunRemove(dockerCli, cmd.Flags(), common.Orchestrator(), opts)
		},
	}
	flags := cmd.Flags()
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}

// RunRemove performs a stack remove against the specified orchestrator
func RunRemove(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.Remove) error {
	if hasServerSideStacks(dockerCli) {
		return runServerSideRemove(dockerCli, commonOrchestrator, opts)
	}
	return runLegacyOrchestratedCommand(dockerCli, flags, commonOrchestrator,
		func() error { return swarm.RunRemove(dockerCli, opts) },
		func(kli *kubernetes.KubeCli) error { return kubernetes.RunRemove(kli, opts) })
}

func runServerSideRemove(dockerCli command.Cli, commonOrchestrator command.Orchestrator, opts options.Remove) error {
	ctx := context.Background()
	dclient := dockerCli.Client()
	stacks := []types.Stack{}
	for _, name := range opts.Namespaces {
		stack, err := getStackByName(ctx, dockerCli, string(commonOrchestrator), name)
		if err != nil {
			return err
		}
		stacks = append(stacks, stack)
	}
	for _, stack := range stacks {
		fmt.Fprintf(dockerCli.Out(), "Removing Stack %s\n", stack.Metadata.Name)
		err := dclient.StackDelete(ctx, stack.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

type stackNotFound string

func (n stackNotFound) NotFound() bool {
	return true
}

func (n stackNotFound) Error() string {
	return fmt.Sprintf("stack %s not found", n)
}

func getStackByName(ctx context.Context, dockerCli command.Cli, orchestrator, name string) (types.Stack, error) {
	dclient := dockerCli.Client()
	filter := filters.NewArgs()
	filter.Add("name", name)
	switch orchestrator {
	case "all":
	case "":
		// No filter needed
	default:
		filter.Add("orchestrator", orchestrator)
	}

	listOpts := types.StackListOptions{
		Filters: filter,
	}
	stacks, err := dclient.StackList(ctx, listOpts)
	if err != nil {
		return types.Stack{}, err
	}

	// TODO - temporary code to workaround broken filters on backend
	// Ultimately we should check for a single item in the list and just return it
	for _, stack := range stacks {
		if stack.Metadata.Name == name {
			return stack, nil
		}
	}
	return types.Stack{}, stackNotFound(name)

}
