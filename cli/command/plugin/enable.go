package plugin

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

func newEnableCommand(dockerCLI command.Cli) *cobra.Command {
	var opts client.PluginEnableOptions

	cmd := &cobra.Command{
		Use:   "enable [OPTIONS] PLUGIN",
		Short: "Enable a plugin",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := runEnable(cmd.Context(), dockerCLI, name, opts); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(dockerCLI.Out(), name)
			return nil
		},
		ValidArgsFunction:     completeNames(dockerCLI, stateDisabled),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.IntVar(&opts.Timeout, "timeout", 30, "HTTP client timeout (in seconds)")
	return cmd
}

func runEnable(ctx context.Context, dockerCli command.Cli, name string, opts client.PluginEnableOptions) error {
	if opts.Timeout < 0 {
		return fmt.Errorf("negative timeout %d is invalid", opts.Timeout)
	}
	_, err := dockerCli.Client().PluginEnable(ctx, name, opts)
	return err
}
