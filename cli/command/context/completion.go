// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package context

import (
	"slices"

	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

type contextProvider interface {
	ContextStore() store.Store
	CurrentContext() string
}

// completeContextNames implements shell completion for context-names.
//
// FIXME(thaJeztah): export, and remove duplicate of this function in cmd/docker.
func completeContextNames(dockerCLI contextProvider, limit int, withFileComp bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if limit > 0 && len(args) >= limit {
			if withFileComp {
				// Provide file/path completion after context name (for "docker context export")
				return nil, cobra.ShellCompDirectiveDefault
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// TODO(thaJeztah): implement function similar to [store.Names] to (also) include descriptions.
		names, _ := store.Names(dockerCLI.ContextStore())
		out := make([]string, 0, len(names))
		for _, name := range names {
			if slices.Contains(args, name) {
				// Already completed
				continue
			}
			if name == dockerCLI.CurrentContext() {
				name += "\tcurrent"
			}
			out = append(out, name)
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}
