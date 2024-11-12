// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package service

import (
	"context"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	refs   []string
	format string
	pretty bool
}

func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] SERVICE [SERVICE...]",
		Short: "Display detailed information on one or more services",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.refs = args

			if opts.pretty && len(opts.format) > 0 {
				return errors.Errorf("--format is incompatible with human friendly format")
			}
			return runInspect(cmd.Context(), dockerCli, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return CompletionFn(dockerCli)(cmd, args, toComplete)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	flags.BoolVar(&opts.pretty, "pretty", false, "Print the information in a human friendly format")
	return cmd
}

func runInspect(ctx context.Context, dockerCli command.Cli, opts inspectOptions) error {
	client := dockerCli.Client()

	if opts.pretty {
		opts.format = "pretty"
	}

	getRef := func(ref string) (any, []byte, error) {
		// Service inspect shows defaults values in empty fields.
		service, _, err := client.ServiceInspectWithRaw(ctx, ref, types.ServiceInspectOptions{InsertDefaults: true})
		if err == nil || !errdefs.IsNotFound(err) {
			return service, nil, err
		}
		return nil, nil, errors.Errorf("Error: no such service: %s", ref)
	}

	getNetwork := func(ref string) (any, []byte, error) {
		nw, _, err := client.NetworkInspectWithRaw(ctx, ref, network.InspectOptions{Scope: "swarm"})
		if err == nil || !errdefs.IsNotFound(err) {
			return nw, nil, err
		}
		return nil, nil, errors.Errorf("Error: no such network: %s", ref)
	}

	f := opts.format
	if len(f) == 0 {
		f = "raw"
		if len(dockerCli.ConfigFile().ServiceInspectFormat) > 0 {
			f = dockerCli.ConfigFile().ServiceInspectFormat
		}
	}

	// check if the user is trying to apply a template to the pretty format, which
	// is not supported
	if strings.HasPrefix(f, "pretty") && f != "pretty" {
		return errors.Errorf("Cannot supply extra formatting options to the pretty template")
	}

	serviceCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(f),
	}

	if err := InspectFormatWrite(serviceCtx, opts.refs, getRef, getNetwork); err != nil {
		return cli.StatusError{StatusCode: 1, Status: err.Error()}
	}
	return nil
}
