package registry

import (
	"context"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	"github.com/spf13/cobra"
)

type searchOptions struct {
	format  string
	term    string
	noTrunc bool
	limit   int
	filter  opts.FilterOpt

	// Deprecated
	stars     uint
	automated bool
}

// NewSearchCommand creates a new `docker search` command
func NewSearchCommand(dockerCli command.Cli) *cobra.Command {
	options := searchOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "search [OPTIONS] TERM",
		Short: "Search the Docker Hub for images",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.term = args[0]
			return runSearch(dockerCli, options)
		},
	}

	flags := cmd.Flags()

	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Don't truncate output")
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")
	flags.IntVar(&options.limit, "limit", registry.DefaultSearchLimit, "Max number of search results")
	flags.StringVar(&options.format, "format", "", "Pretty-print search using a Go template")

	flags.BoolVar(&options.automated, "automated", false, "Only show automated builds")
	flags.UintVarP(&options.stars, "stars", "s", 0, "Only displays with at least x stars")

	flags.MarkDeprecated("automated", "use --filter=is-automated=true instead")
	flags.MarkDeprecated("stars", "use --filter=stars=3 instead")

	return cmd
}

func runSearch(dockerCli command.Cli, options searchOptions) error {
	indexInfo, err := registry.ParseSearchIndexInfo(options.term)
	if err != nil {
		return err
	}

	ctx := context.Background()

	authConfig := command.ResolveAuthConfig(ctx, dockerCli, indexInfo)
	requestPrivilege := command.RegistryAuthenticationPrivilegedFunc(dockerCli, indexInfo, "search")

	encodedAuth, err := command.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}

	searchOptions := types.ImageSearchOptions{
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: requestPrivilege,
		Filters:       options.filter.Value(),
		Limit:         options.limit,
	}

	clnt := dockerCli.Client()

	results, err := clnt.ImageSearch(ctx, options.term, searchOptions)
	if err != nil {
		return err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[j].StarCount < results[i].StarCount
	})
	searchCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewSearchFormat(options.format),
		Trunc:  !options.noTrunc,
	}
	return formatter.SearchWrite(searchCtx, results, options.automated, int(options.stars))
}
