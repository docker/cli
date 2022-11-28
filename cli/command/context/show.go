package context

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/spf13/cobra"
)

// newShowCommand creates a new cobra.Command for `docker context sow`
func newShowCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print the name of the current context",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runShow(dockerCli)
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return cmd
}

func runShow(dockerCli command.Cli) {
	fmt.Fprintln(dockerCli.Out(), dockerCli.CurrentContext())
}
