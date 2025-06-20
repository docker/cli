package registry

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/opts"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/registry"
	"github.com/spf13/cobra"
)

type searchOptions struct {
	format  string
	term    string
	noTrunc bool
	limit   int
	filter  opts.FilterOpt
}

// NewSearchCommand creates a new `docker search` command
func NewSearchCommand(dockerCli command.Cli) *cobra.Command {
	options := searchOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "search [OPTIONS] TERM",
		Short: "Search Docker Hub for images",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.term = args[0]
			return runSearch(cmd.Context(), dockerCli, options)
		},
		Annotations: map[string]string{
			"category-top": "10",
		},
	}

	flags := cmd.Flags()

	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Don't truncate output")
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")
	flags.IntVar(&options.limit, "limit", 0, "Max number of search results")
	flags.StringVar(&options.format, "format", "", "Pretty-print search using a Go template")

	return cmd
}

func runSearch(ctx context.Context, dockerCli command.Cli, options searchOptions) error {
	if options.filter.Value().Contains("is-automated") {
		_, _ = fmt.Fprintln(dockerCli.Err(), `WARNING: the "is-automated" filter is deprecated, and searching for "is-automated=true" will not yield any results in future.`)
	}
	indexInfo, err := registry.ParseSearchIndexInfo(options.term)
	if err != nil {
		return err
	}

	authConfig := command.ResolveAuthConfig(dockerCli.ConfigFile(), indexInfo)
	encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
	if err != nil {
		return err
	}

	var requestPrivilege registrytypes.RequestAuthConfig
	if dockerCli.In().IsTerminal() {
		requestPrivilege = command.RegistryAuthenticationPrivilegedFunc(dockerCli, indexInfo, "search")
	}
	results, err := dockerCli.Client().ImageSearch(ctx, options.term, registrytypes.SearchOptions{
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: requestPrivilege,
		Filters:       options.filter.Value(),
		Limit:         options.limit,
	})
	if err != nil {
		return err
	}

	searchCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewSearchFormat(options.format),
		Trunc:  !options.noTrunc,
	}
	return SearchWrite(searchCtx, results)
}
