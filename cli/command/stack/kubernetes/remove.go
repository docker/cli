package kubernetes

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type removeOptions struct {
	stacks []string
}

func newRemoveCommand(dockerCli command.Cli, kubeCli *kubeCli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm STACK [STACK...]",
		Aliases: []string{"remove", "down"},
		Short:   "Remove one or more stacks",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stacks = args
			return runRemove(dockerCli, kubeCli, opts)
		},
	}
	return cmd
}

func runRemove(dockerCli command.Cli, kubeCli *kubeCli, opts removeOptions) error {
	stacks, err := kubeCli.Stacks()
	if err != nil {
		return err
	}
	for _, stack := range opts.stacks {
		fmt.Fprintf(dockerCli.Out(), "Removing stack: %s\n", stack)
		err := stacks.Delete(stack, &metav1.DeleteOptions{})
		if err != nil {
			fmt.Fprintf(dockerCli.Out(), "Failed to remove stack %s: %s\n", stack, err)
			return err
		}
	}
	return nil
}
