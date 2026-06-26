package secret

import (
	"context"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/fvbommel/sortorder"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type listOptions struct {
	quiet  bool
	format string
	filter opts.FilterOpt
}

func newSecretListCommand(dockerCLI command.Cli) *cobra.Command {
	options := listOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List secrets",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecretList(cmd.Context(), dockerCLI, options)
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

func runSecretList(ctx context.Context, dockerCLI command.Cli, options listOptions) error {
	apiClient := dockerCLI.Client()

	res, err := apiClient.SecretList(ctx, client.SecretListOptions{Filters: options.filter.Value()})
	if err != nil {
		return err
	}
	format := options.format
	if len(format) == 0 {
		if len(dockerCLI.ConfigFile().SecretFormat) > 0 && !options.quiet {
			format = dockerCLI.ConfigFile().SecretFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	sort.Slice(res.Items, func(i, j int) bool {
		return sortorder.NaturalLess(res.Items[i].Spec.Name, res.Items[j].Spec.Name)
	})

	secretCtx := formatter.Context{
		Output: dockerCLI.Out(),
		Format: newFormat(format, options.quiet),
	}
	return formatWrite(secretCtx, res)
}
