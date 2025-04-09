// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package image

import (
	"bytes"
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/inspect"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
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
		ValidArgsFunction: completion.ImageNames(dockerCli, -1),
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	return cmd
}

func runInspect(ctx context.Context, dockerCLI command.Cli, opts inspectOptions) error {
	apiClient := dockerCLI.Client()
	return inspect.Inspect(dockerCLI.Out(), opts.refs, opts.format, func(ref string) (any, []byte, error) {
		var buf bytes.Buffer
		resp, err := apiClient.ImageInspect(ctx, ref, client.ImageInspectWithRawResponse(&buf))
		if err != nil {
			return image.InspectResponse{}, nil, err
		}
		return resp, buf.Bytes(), err
	})
}
