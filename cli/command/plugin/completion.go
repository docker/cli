package plugin

import (
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type pluginState string

const (
	stateAny      pluginState = ""
	stateEnabled  pluginState = "enabled"
	stateDisabled pluginState = "disabled"
)

// completeNames offers completion for plugin names in the given state.
// The state argument can be one of:
//
// - "all": all plugins
// - "enabled": all enabled plugins
// - "disabled": all disabled plugins
func completeNames(dockerCLI completion.APIClientProvider, state pluginState) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		f := make(client.Filters)
		switch state {
		case stateEnabled:
			f.Add("enabled", "true")
		case stateDisabled:
			f.Add("enabled", "false")
		case stateAny:
			// no filter
		}

		res, err := dockerCLI.Client().PluginList(cmd.Context(), client.PluginListOptions{
			Filters: f,
		})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, v := range res.Items {
			names = append(names, v.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
