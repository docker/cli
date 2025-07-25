package plugin

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newEnableCommand(dockerCli command.Cli) *cobra.Command {
	var opts client.PluginEnableOptions

	cmd := &cobra.Command{
		Use:   "enable [OPTIONS] PLUGIN",
		Short: "Enable a plugin",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := runEnable(cmd.Context(), dockerCli, name, opts); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(dockerCli.Out(), name)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.IntVar(&opts.Timeout, "timeout", 30, "HTTP client timeout (in seconds)")
	return cmd
}

func runEnable(ctx context.Context, dockerCli command.Cli, name string, opts client.PluginEnableOptions) error {
	if opts.Timeout < 0 {
		return errors.Errorf("negative timeout %d is invalid", opts.Timeout)
	}
	return dockerCli.Client().PluginEnable(ctx, name, opts)
}
