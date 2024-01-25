// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package secret

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
	names  []string
	format string
	pretty bool
}

func newSecretInspectCommand(dockerCli command.Cli) *cobra.Command {
	opts := inspectOptions{}
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] SECRET [SECRET...]",
		Short: "Display detailed information on one or more secrets",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runSecretInspect(dockerCli, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCli)(cmd, args, toComplete)
		},
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "Print the information in a human friendly format")
	return cmd
}

func runSecretInspect(dockerCli command.Cli, opts inspectOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	if opts.pretty {
		opts.format = "pretty"
	}

	getRef := func(id string) (interface{}, []byte, error) {
		return client.SecretInspectWithRaw(ctx, id)
	}
	f := opts.format

	// check if the user is trying to apply a template to the pretty format, which
	// is not supported
	if strings.HasPrefix(f, "pretty") && f != "pretty" {
		return errors.New("cannot supply extra formatting options to the pretty template")
	}

	secretCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(f, false),
	}

	if err := InspectFormatWrite(secretCtx, opts.names, getRef); err != nil {
		return cli.StatusError{StatusCode: 1, Status: err.Error()}
	}
	return nil
}
