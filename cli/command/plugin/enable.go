package plugin

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type enableOpts struct {
	timeout int
	name    string
}

func newEnableCommand(dockerCli command.Cli) *cobra.Command {
	var opts enableOpts

	cmd := &cobra.Command{
		Use:   "enable [OPTIONS] PLUGIN",
		Short: "Enable a plugin",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return runEnable(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.IntVar(&opts.timeout, "timeout", 30, "HTTP client timeout (in seconds)")
	return cmd
}

func runEnable(dockerCli command.Cli, opts *enableOpts) error {
	name := opts.name
	if opts.timeout < 0 {
		return errors.Errorf("negative timeout %d is invalid", opts.timeout)
	}

	if err := dockerCli.Client().PluginEnable(context.Background(), name, types.PluginEnableOptions{Timeout: opts.timeout}); err != nil {
		return err
	}
	fmt.Fprintln(dockerCli.Out(), name)
	return nil
}
