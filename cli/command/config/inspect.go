package config

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type inspectOptions struct {
	names  []string
	format string
}

func newConfigInspectCommand(dockerCli command.Cli) *cobra.Command {
	opts := inspectOptions{}
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] CONFIG [CONFIG...]",
		Short: "Display detailed information on one or more configuration files",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.names = args
			return runConfigInspect(dockerCli, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", "Format the output using the given Go template")
	return cmd
}

func runConfigInspect(dockerCli command.Cli, opts inspectOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	getRef := func(id string) (interface{}, []byte, error) {
		return client.ConfigInspectWithRaw(ctx, id)
	}

	return inspect.Inspect(dockerCli.Out(), opts.names, opts.format, getRef)
}
