// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package volume

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/inspect"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	format string
	names  []string
}

func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] VOLUME [VOLUME...]",
		Short: "Display detailed information on one or more volumes",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runInspect(cmd.Context(), dockerCli, opts)
		},
		ValidArgsFunction: completion.VolumeNames(dockerCli),
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)

	return cmd
}

func runInspect(ctx context.Context, dockerCli command.Cli, opts inspectOptions) error {
	client := dockerCli.Client()

	getVolFunc := func(name string) (any, []byte, error) {
		i, err := client.VolumeInspect(ctx, name)
		return i, nil, err
	}

	return inspect.Inspect(dockerCli.Out(), opts.names, opts.format, getVolFunc)
}
