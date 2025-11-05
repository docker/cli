// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	refs   []string
	format string
	pretty bool
}

func newInspectCommand(dockerCLI command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] SERVICE [SERVICE...]",
		Short: "Display detailed information on one or more services",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.refs = args

			if opts.pretty && len(opts.format) > 0 {
				return errors.New("--format is incompatible with human friendly format")
			}
			return runInspect(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction:     completeServiceNames(dockerCLI),
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
		// Service inspect shows defaults values in empty fields.
		res, err := apiClient.ServiceInspect(ctx, ref, client.ServiceInspectOptions{InsertDefaults: true})
		if err == nil || !errdefs.IsNotFound(err) {
			return res.Service, res.Raw, err
		}
		return nil, nil, fmt.Errorf("no such service: %s", ref)
	}

	getNetwork := func(ref string) (any, []byte, error) {
		res, err := apiClient.NetworkInspect(ctx, ref, client.NetworkInspectOptions{Scope: "swarm"})
		if err == nil || !errdefs.IsNotFound(err) {
			return res.Network, res.Raw, err
		}
		return nil, nil, fmt.Errorf("no such network: %s", ref)
	}

	f := opts.format
	if len(f) == 0 {
		f = "raw"
		if len(dockerCLI.ConfigFile().ServiceInspectFormat) > 0 {
			f = dockerCLI.ConfigFile().ServiceInspectFormat
		}
	}

	// check if the user is trying to apply a template to the pretty format, which
	// is not supported
	if strings.HasPrefix(f, "pretty") && f != "pretty" {
		return errors.New("cannot supply extra formatting options to the pretty template")
	}

	serviceCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(f),
	}

	if err := inspectFormatWrite(serviceCtx, opts.refs, getRef, getNetwork); err != nil {
		return cli.StatusError{StatusCode: 1, Status: err.Error()}
	}
	return nil
}
