package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/commands"
	"github.com/docker/cli/opts"
	registrytypes "github.com/moby/moby/api/types/registry"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newSearchCommand)
}

type searchOptions struct {
	format  string
	term    string
	noTrunc bool
	limit   int
	filter  opts.FilterOpt
}

// newSearchCommand creates a new `docker search` command
func newSearchCommand(dockerCLI command.Cli) *cobra.Command {
	options := searchOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "search [OPTIONS] TERM",
		Short: "Search Docker Hub for images",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.term = args[0]
			return runSearch(cmd.Context(), dockerCLI, options)
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
	encodedAuth, err := getAuth(dockerCli, options.term)
	if err != nil {
		return err
	}

	results, err := dockerCli.Client().ImageSearch(ctx, options.term, registrytypes.SearchOptions{
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: nil,
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

// authConfigKey is the key used to store credentials for Docker Hub. It is
// a copy of [registry.IndexServer].
//
// [registry.IndexServer]: https://pkg.go.dev/github.com/docker/docker/registry#IndexServer
const authConfigKey = "https://index.docker.io/v1/"

// getAuth will use fetch auth based on the given search-term. If the search
// does not contain a hostname for the registry, it assumes Docker Hub is used,
// and resolves authentication for Docker Hub, otherwise it resolves authentication
// for the given registry.
func getAuth(dockerCLI command.Cli, reposName string) (encodedAuth string, err error) {
	authCfgKey := splitReposSearchTerm(reposName)
	if authCfgKey == "docker.io" || authCfgKey == "index.docker.io" {
		authCfgKey = authConfigKey
	}

	// Ignoring errors here, which was the existing behavior (likely
	// "no credentials found"). We'll get an error when search failed,
	// so fine to ignore in most situations.
	authConfig, _ := dockerCLI.ConfigFile().GetAuthConfig(authCfgKey)
	return registrytypes.EncodeAuthConfig(registrytypes.AuthConfig(authConfig))
}

// splitReposSearchTerm breaks a search term into an index name and remote name
func splitReposSearchTerm(reposName string) string {
	nameParts := strings.SplitN(reposName, "/", 2)
	if len(nameParts) == 1 || (!strings.Contains(nameParts[0], ".") && !strings.Contains(nameParts[0], ":") && nameParts[0] != "localhost") {
		// This is a Docker Hub repository (ex: samalba/hipache or ubuntu),
		// use the default Docker Hub registry (docker.io)
		return "docker.io"
	}
	return nameParts[0]
}
