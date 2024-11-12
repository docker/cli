// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package image

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
	refs   []string
}

// newInspectCommand creates a new cobra.Command for `docker image inspect`
func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] IMAGE [IMAGE...]",
		Short: "Display detailed information on one or more images",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.refs = args
			return runInspect(cmd.Context(), dockerCli, opts)
		},
		ValidArgsFunction: completion.ImageNames(dockerCli),
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	return cmd
}

func runInspect(ctx context.Context, dockerCli command.Cli, opts inspectOptions) error {
	client := dockerCli.Client()
	getRefFunc := func(ref string) (any, []byte, error) {
		return client.ImageInspectWithRaw(ctx, ref)
	}
	return inspect.Inspect(dockerCli.Out(), opts.refs, opts.format, getRefFunc)
}
