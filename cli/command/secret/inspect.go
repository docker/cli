// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package secret

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
	names  []string
	format string
	pretty bool
}

func newSecretInspectCommand(dockerCLI command.Cli) *cobra.Command {
	opts := inspectOptions{}
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] SECRET [SECRET...]",
		Short: "Display detailed information on one or more secrets",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runSecretInspect(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction:     completeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "Print the information in a human friendly format")
	return cmd
}

func runSecretInspect(ctx context.Context, dockerCLI command.Cli, opts inspectOptions) error {
	apiClient := dockerCLI.Client()

	if opts.pretty {
		opts.format = "pretty"
	}

	getRef := func(id string) (any, []byte, error) {
		res, err := apiClient.SecretInspect(ctx, id, client.SecretInspectOptions{})
		return res.Secret, res.Raw, err
	}

	// check if the user is trying to apply a template to the pretty format, which
	// is not supported
	if strings.HasPrefix(opts.format, "pretty") && opts.format != "pretty" {
		return errors.New("cannot supply extra formatting options to the pretty template")
	}

	secretCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(opts.format, false),
	}

	if err := inspectFormatWrite(secretCtx, opts.names, getRef); err != nil {
		return cli.StatusError{StatusCode: 1, Status: err.Error()}
	}
	return nil
}
