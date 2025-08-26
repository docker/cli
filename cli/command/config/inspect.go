// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

package config

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

// InspectOptions contains options for the docker config inspect command.
//
// Deprecated: this type was for internal use and will be removed in the next release.
type InspectOptions struct {
	Names  []string
	Format string
	Pretty bool
}

// inspectOptions contains options for the docker config inspect command.
type inspectOptions struct {
	names  []string
	format string
	pretty bool
}

func newConfigInspectCommand(dockerCLI command.Cli) *cobra.Command {
	opts := inspectOptions{}
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] CONFIG [CONFIG...]",
		Short: "Display detailed information on one or more configs",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runInspect(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCLI)(cmd, args, toComplete)
		},
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "Print the information in a human friendly format")
	return cmd
}

// RunConfigInspect inspects the given Swarm config.
//
// Deprecated: this function was for internal use and will be removed in the next release.
func RunConfigInspect(ctx context.Context, dockerCLI command.Cli, opts InspectOptions) error {
	return runInspect(ctx, dockerCLI, inspectOptions{
		names:  opts.Names,
		format: opts.Format,
		pretty: opts.Pretty,
	})
}

// runInspect inspects the given Swarm config.
func runInspect(ctx context.Context, dockerCLI command.Cli, opts inspectOptions) error {
	apiClient := dockerCLI.Client()

	if opts.pretty {
		opts.format = "pretty"
	}

	getRef := func(id string) (any, []byte, error) {
		return apiClient.ConfigInspectWithRaw(ctx, id)
	}

	// check if the user is trying to apply a template to the pretty format, which
	// is not supported
	if strings.HasPrefix(opts.format, "pretty") && opts.format != "pretty" {
		return errors.New("cannot supply extra formatting options to the pretty template")
	}

	configCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(opts.format, false),
	}

	if err := inspectFormatWrite(configCtx, opts.names, getRef); err != nil {
		return cli.StatusError{StatusCode: 1, Status: err.Error()}
	}
	return nil
}
