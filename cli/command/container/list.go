package container

import (
	"context"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/docker/cli/templates"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type psOptions struct {
	quiet       bool
	size        bool
	sizeChanged bool
	all         bool
	noTrunc     bool
	nLatest     bool
	last        int
	format      string
	filter      opts.FilterOpt
}

// NewPsCommand creates a new cobra.Command for `docker ps`
func NewPsCommand(dockerCli command.Cli) *cobra.Command {
	options := psOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS]",
		Short: "List containers",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.sizeChanged = cmd.Flags().Changed("size")
			return runPs(dockerCli, &options)
		},
		Annotations: map[string]string{
			"category-top": "3",
			"aliases":      "docker container ls, docker container list, docker container ps, docker ps",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()

	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display container IDs")
	flags.BoolVarP(&options.size, "size", "s", false, "Display total file sizes")
	flags.BoolVarP(&options.all, "all", "a", false, "Show all containers (default shows just running)")
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Don't truncate output")
	flags.BoolVarP(&options.nLatest, "latest", "l", false, "Show the latest created container (includes all states)")
	flags.IntVarP(&options.last, "last", "n", -1, "Show n last created containers (includes all states)")
	flags.StringVarP(&options.format, "format", "", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

func newListCommand(dockerCli command.Cli) *cobra.Command {
	cmd := *NewPsCommand(dockerCli)
	cmd.Aliases = []string{"ps", "list"}
	cmd.Use = "ls [OPTIONS]"
	return &cmd
}

func buildContainerListOptions(opts *psOptions) (*types.ContainerListOptions, error) {
	options := &types.ContainerListOptions{
		All:     opts.all,
		Limit:   opts.last,
		Size:    opts.size,
		Filters: opts.filter.Value(),
	}

	if opts.nLatest && opts.last == -1 {
		options.Limit = 1
	}

	// always validate template when `--format` is used, for consistency
	if len(opts.format) > 0 {
		tmpl, err := templates.NewParse("", opts.format)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse template")
		}

		optionsProcessor := formatter.NewContainerContext()

		// This shouldn't error out but swallowing the error makes it harder
		// to track down if preProcessor issues come up.
		if err := tmpl.Execute(io.Discard, optionsProcessor); err != nil {
			return nil, errors.Wrap(err, "failed to execute template")
		}

		// if `size` was not explicitly set to false (with `--size=false`)
		// and `--quiet` is not set, request size if the template requires it
		if !opts.quiet && !options.Size && !opts.sizeChanged {
			// The --size option isn't set, but .Size may be used in the template.
			// Parse and execute the given template to detect if the .Size field is
			// used. If it is, then automatically enable the --size option. See #24696
			//
			// Only requesting container size information when needed is an optimization,
			// because calculating the size is a costly operation.

			if _, ok := optionsProcessor.FieldsUsed["Size"]; ok {
				options.Size = true
			}
		}
	}

	return options, nil
}

func runPs(dockerCli command.Cli, options *psOptions) error {
	ctx := context.Background()

	if len(options.format) == 0 {
		// load custom psFormat from CLI config (if any)
		options.format = dockerCli.ConfigFile().PsFormat
	}

	listOptions, err := buildContainerListOptions(options)
	if err != nil {
		return err
	}

	containers, err := dockerCli.Client().ContainerList(ctx, *listOptions)
	if err != nil {
		return err
	}

	containerCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewContainerFormat(options.format, options.quiet, listOptions.Size),
		Trunc:  !options.noTrunc,
	}
	return formatter.ContainerWrite(containerCtx, containers)
}
