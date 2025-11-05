// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package node

import (
	"context"
	"errors"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	nodeIds []string
	format  string
	pretty  bool
}

func newInspectCommand(dockerCLI command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] self|NODE [NODE...]",
		Short: "Display detailed information on one or more nodes",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.nodeIds = args
			return runInspect(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction:     completeNodeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	flags.BoolVar(&opts.pretty, "pretty", false, "Print the information in a human friendly format")
	return cmd
}

func runInspect(ctx context.Context, dockerCLI command.Cli, opts inspectOptions) error {
	apiClient := dockerCLI.Client()

	if opts.pretty {
		opts.format = "pretty"
	}

	getRef := func(ref string) (any, []byte, error) {
		nodeRef, err := Reference(ctx, apiClient, ref)
		if err != nil {
			return nil, nil, err
		}
		res, err := apiClient.NodeInspect(ctx, nodeRef, client.NodeInspectOptions{})
		return res.Node, res.Raw, err
	}

	// check if the user is trying to apply a template to the pretty format, which
	// is not supported
	if strings.HasPrefix(opts.format, "pretty") && opts.format != "pretty" {
		return errors.New("cannot supply extra formatting options to the pretty template")
	}

	nodeCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(opts.format, false),
	}

	if err := inspectFormatWrite(nodeCtx, opts.nodeIds, getRef); err != nil {
		return cli.StatusError{StatusCode: 1, Status: err.Error()}
	}
	return nil
}
