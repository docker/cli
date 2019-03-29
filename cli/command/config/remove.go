package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RemoveOptions contains options for the docker config rm command.
type RemoveOptions struct {
	Names []string
}

func newConfigRemoveCommand(dockerCli command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:     "rm CONFIG [CONFIG...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more configs",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := RemoveOptions{
				Names: args,
			}
			return RunConfigRemove(dockerCli, opts)
		},
	}
}

// RunConfigRemove removes the given Swarm configs.
func RunConfigRemove(dockerCli command.Cli, opts RemoveOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	var errs []string

	for _, name := range opts.Names {
		if err := client.ConfigRemove(ctx, name); err != nil {
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
