// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package plugin

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/inspect"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	pluginNames []string
	format      string
}

func newInspectCommand(dockerCLI command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] PLUGIN [PLUGIN...]",
		Short: "Display detailed information on one or more plugins",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.pluginNames = args
			return runInspect(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction:     completeNames(dockerCLI, stateAny),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	return cmd
}

func runInspect(ctx context.Context, dockerCLI command.Cli, opts inspectOptions) error {
	apiClient := dockerCLI.Client()
	return inspect.Inspect(dockerCLI.Out(), opts.pluginNames, opts.format, func(ref string) (any, []byte, error) {
		res, err := apiClient.PluginInspect(ctx, ref, client.PluginInspectOptions{})
		return res.Plugin, res.Raw, err
	})
}
