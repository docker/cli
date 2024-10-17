package system

import (
	"strings"

	"github.com/docker/cli/cli/command/completion"
	"github.com/spf13/cobra"
)

var (
	eventFilters = []string{"container", "daemon", "event", "image", "label", "network", "node", "scope", "type", "volume"}
	eventNames   = []string{
		"attach",
		"commit",
		"connect",
		"copy",
		"create",
		"delete",
		"destroy",
		"detach",
		"die",
		"disable",
		"disconnect",
		"enable",
		"exec_create",
		"exec_detach",
		"exec_die",
		"exec_start",
		"export",
		"health_status",
		"import",
		"install",
		"kill",
		"load",
		"mount",
		"oom",
		"pause",
		"pull",
		"push",
		"reload",
		"remove",
		"rename",
		"resize",
		"restart",
		"save",
		"start",
		"stop",
		"tag",
		"top",
		"unmount",
		"unpause",
		"untag",
		"update",
	}
	eventTypes = []string{"config", "container", "daemon", "image", "network", "node", "plugin", "secret", "service", "volume"}
)

func completeFilters(dockerCLI completion.APIClientProvider) completion.ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if strings.HasPrefix(toComplete, "container=") {
			names, _ := completion.ContainerNames(dockerCLI, true)(cmd, args, toComplete)
			if names == nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return prefixWith("container=", names), cobra.ShellCompDirectiveDefault
		}
		if strings.HasPrefix(toComplete, "event=") {
			return prefixWith("event=", eventNames), cobra.ShellCompDirectiveDefault
		}
		if strings.HasPrefix(toComplete, "label=") {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if strings.HasPrefix(toComplete, "network=") {
			names, _ := completion.NetworkNames(dockerCLI)(cmd, args, toComplete)
			if names == nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return prefixWith("network=", names), cobra.ShellCompDirectiveDefault
		}
		if strings.HasPrefix(toComplete, "type=") {
			return prefixWith("type=", eventTypes), cobra.ShellCompDirectiveDefault
		}
		return postfixWith("=", eventFilters), cobra.ShellCompDirectiveNoSpace
	}
}

func prefixWith(prefix string, values []string) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = prefix + v
	}
	return result
}

func postfixWith(postfix string, values []string) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v + postfix
	}
	return result
}
