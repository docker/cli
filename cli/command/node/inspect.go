// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package node

import (
	"context"
	"errors"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	nodeIds []string
	format  string
	pretty  bool
}

func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] self|NODE [NODE...]",
		Short: "Display detailed information on one or more nodes",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.nodeIds = args
			return runInspect(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	flags.BoolVar(&opts.pretty, "pretty", false, "Print the information in a human friendly format")
	return cmd
}

func runInspect(dockerCli command.Cli, opts inspectOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	if opts.pretty {
		opts.format = "pretty"
	}

	getRef := func(ref string) (interface{}, []byte, error) {
		nodeRef, err := Reference(ctx, client, ref)
		if err != nil {
			return nil, nil, err
		}
		node, _, err := client.NodeInspectWithRaw(ctx, nodeRef)
		return node, nil, err
	}
	f := opts.format

	// check if the user is trying to apply a template to the pretty format, which
	// is not supported
	if strings.HasPrefix(f, "pretty") && f != "pretty" {
		return errors.New("cannot supply extra formatting options to the pretty template")
	}

	nodeCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(f, false),
	}

	if err := InspectFormatWrite(nodeCtx, opts.nodeIds, getRef); err != nil {
		return cli.StatusError{StatusCode: 1, Status: err.Error()}
	}
	return nil
}
