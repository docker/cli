package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm SERVICE [SERVICE...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more services",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCli, args)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return CompletionFn(dockerCli)(cmd, args, toComplete)
		},
	}
	cmd.Flags()

	return cmd
}

func runRemove(ctx context.Context, dockerCli command.Cli, sids []string) error {
	client := dockerCli.Client()

	var errs []string
	for _, sid := range sids {
		err := client.ServiceRemove(ctx, sid)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		_, _ = fmt.Fprintf(dockerCli.Out(), "%s\n", sid)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
