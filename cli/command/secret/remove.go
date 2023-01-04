package secret

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	names []string
}

func newSecretRemoveCommand(dockerCli command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:     "rm SECRET [SECRET...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more secrets",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return command.RunSwarm(dockerCli)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCli)(cmd, args, toComplete)
		},
	}
}

func runSecretRemove(dockerCli command.Cli, opts removeOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	var errs []string

	for _, name := range opts.names {
		if err := client.SecretRemove(ctx, name); err != nil {
			errs = append(errs, err.Error())
			continue
		}

		fmt.Fprintln(dockerCli.Out(), name)
	}

	if len(errs) > 0 {
		return errors.Errorf("%s", strings.Join(errs, "\n"))
	}

	return nil
}
