package service

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type listOptions struct {
	quiet  bool
	format string
	filter opts.FilterOpt
}

func newListCommand(dockerCLI command.Cli) *cobra.Command {
	options := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List services",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

func runList(ctx context.Context, dockerCLI command.Cli, options listOptions) error {
	apiClient := dockerCLI.Client()
	res, err := apiClient.ServiceList(ctx, client.ServiceListOptions{
		Filters: options.filter.Value(),
		// When not running "quiet", also get service status (number of running
		// and desired tasks). Note that this is only supported on API v1.41 and
		// up; older API versions ignore this option, and we will have to collect
		// the information manually below.
		Status: !options.quiet,
	})
	if err != nil {
		return err
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCLI.ConfigFile().ServicesFormat) > 0 && !options.quiet {
			format = dockerCLI.ConfigFile().ServicesFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	servicesCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: NewListFormat(format, options.quiet),
	}
	return ListFormatWrite(servicesCtx, res)
}
