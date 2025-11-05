// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package network

import (
	"context"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/inspect"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	format  string
	names   []string
	verbose bool
}

func newInspectCommand(dockerCLI command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] NETWORK [NETWORK...]",
		Short: "Display detailed information on one or more networks",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runInspect(cmd.Context(), dockerCLI.Client(), dockerCLI.Out(), opts)
		},
		ValidArgsFunction:     completion.NetworkNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Verbose output for diagnostics")

	return cmd
}

func runInspect(ctx context.Context, apiClient client.NetworkAPIClient, output io.Writer, opts inspectOptions) error {
	return inspect.Inspect(output, opts.names, opts.format, func(name string) (any, []byte, error) {
		res, err := apiClient.NetworkInspect(ctx, name, client.NetworkInspectOptions{Verbose: opts.verbose})
		if err != nil {
			return nil, nil, err
		}
		return res.Network, res.Raw, nil
	})
}
