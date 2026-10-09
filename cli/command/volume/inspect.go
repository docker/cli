package volume

import (
	"context"

	"github.com/docker/cli/v29/cli"
	"github.com/docker/cli/v29/cli/command"
	"github.com/docker/cli/v29/cli/command/completion"
	"github.com/docker/cli/v29/cli/command/inspect"
	flagsHelper "github.com/docker/cli/v29/cli/flags"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	format string
	names  []string
}

func newInspectCommand(dockerCLI command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] VOLUME [VOLUME...]",
		Short: "Display detailed information on one or more volumes",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runInspect(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction:     completion.VolumeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)

	return cmd
}

func runInspect(ctx context.Context, dockerCLI command.Cli, opts inspectOptions) error {
	apiClient := dockerCLI.Client()
	return inspect.Inspect(dockerCLI.Out(), opts.names, opts.format, func(name string) (any, []byte, error) {
		res, err := apiClient.VolumeInspect(ctx, name, client.VolumeInspectOptions{})
		return res.Volume, res.Raw, err
	})
}
