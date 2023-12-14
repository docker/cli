// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

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
type InspectOptions struct {
	Names  []string
	Format string
	Pretty bool
}

func newConfigInspectCommand(dockerCli command.Cli) *cobra.Command {
	opts := InspectOptions{}
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] CONFIG [CONFIG...]",
		Short: "Display detailed information on one or more configs",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Names = args
			return RunConfigInspect(dockerCli, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCli)(cmd, args, toComplete)
		},
	}

	cmd.Flags().StringVarP(&opts.Format, "format", "f", "", flagsHelper.InspectFormatHelp)
	cmd.Flags().BoolVar(&opts.Pretty, "pretty", false, "Print the information in a human friendly format")
	return cmd
}

// RunConfigInspect inspects the given Swarm config.
func RunConfigInspect(dockerCli command.Cli, opts InspectOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	if opts.Pretty {
		opts.Format = "pretty"
	}

	getRef := func(id string) (interface{}, []byte, error) {
		return client.ConfigInspectWithRaw(ctx, id)
	}
	f := opts.Format

	// check if the user is trying to apply a template to the pretty format, which
	// is not supported
	if strings.HasPrefix(f, "pretty") && f != "pretty" {
		return errors.New("cannot supply extra formatting options to the pretty template")
	}

	configCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(f, false),
	}

	if err := InspectFormatWrite(configCtx, opts.Names, getRef); err != nil {
		return cli.StatusError{StatusCode: 1, Status: err.Error()}
	}
	return nil
}
