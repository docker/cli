package service

import (
	"os"
	"strings"

	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

var (
	// serviceListFilters are the filters that can be used with "docker service ls --filter".
	serviceListFilters = []string{"id", "label", "mode", "name"}

	// serviceModes are the valid values for the "mode" filter of "docker service ls".
	serviceModes = []string{"replicated", "global", "replicated-job", "global-job"}

	// servicePsFilters are the filters that can be used with "docker service ps --filter".
	servicePsFilters = []string{"desired-state", "id", "name", "node"}

	// taskDesiredStates are the valid values for the "desired-state" task filter.
	taskDesiredStates = []string{
		string(swarm.TaskStateRunning),
		string(swarm.TaskStateShutdown),
		string(swarm.TaskStateAccepted),
	}
)

// completeServiceNames offers completion for swarm service names and optional IDs.
// By default, only names are returned.
// Set DOCKER_COMPLETION_SHOW_SERVICE_IDS=yes to also complete IDs.
func completeServiceNames(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
	// https://github.com/docker/cli/blob/f9ced58158d5e0b358052432244b483774a1983d/contrib/completion/bash/docker#L41-L43
	showIDs := os.Getenv("DOCKER_COMPLETION_SHOW_SERVICE_IDS") == "yes"
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		res, err := dockerCLI.Client().ServiceList(cmd.Context(), client.ServiceListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := make([]string, 0, len(res.Items))
		for _, service := range res.Items {
			if showIDs {
				names = append(names, service.Spec.Name, service.ID)
			} else {
				names = append(names, service.Spec.Name)
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeServiceListFilters provides completion for the filters that can be
// used with "docker service ls --filter".
func completeServiceListFilters(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		key, _, ok := strings.Cut(toComplete, "=")
		if !ok {
			return completion.WithSuffix("=", serviceListFilters), cobra.ShellCompDirectiveNoSpace
		}
		switch key {
		case "id", "name":
			return completion.WithPrefix(key+"=", serviceNames(dockerCLI, cmd)), cobra.ShellCompDirectiveNoFileComp
		case "mode":
			return completion.WithPrefix("mode=", serviceModes), cobra.ShellCompDirectiveNoFileComp
		case "label":
			return nil, cobra.ShellCompDirectiveNoFileComp
		default:
			return completion.WithSuffix("=", serviceListFilters), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
	}
}

// completeServicePsFilters provides completion for the filters that can be
// used with "docker service ps --filter".
func completeServicePsFilters(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		key, _, ok := strings.Cut(toComplete, "=")
		if !ok {
			return completion.WithSuffix("=", servicePsFilters), cobra.ShellCompDirectiveNoSpace
		}
		switch key {
		case "desired-state":
			return completion.WithPrefix("desired-state=", taskDesiredStates), cobra.ShellCompDirectiveNoFileComp
		case "node":
			return completion.WithPrefix("node=", nodeNames(dockerCLI, cmd)), cobra.ShellCompDirectiveNoFileComp
		case "id", "name":
			// Task IDs and names are not easily discoverable; only offer the key.
			return nil, cobra.ShellCompDirectiveNoFileComp
		default:
			return completion.WithSuffix("=", servicePsFilters), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
	}
}

// serviceNames contacts the API to get a list of service names.
// In case of an error, an empty list is returned.
func serviceNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().ServiceList(cmd.Context(), client.ServiceListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(res.Items))
	for _, service := range res.Items {
		names = append(names, service.Spec.Name)
	}
	return names
}

// nodeNames contacts the API to get a list of node (host)names.
// In case of an error, an empty list is returned.
func nodeNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().NodeList(cmd.Context(), client.NodeListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(res.Items))
	for _, node := range res.Items {
		names = append(names, node.Description.Hostname)
	}
	return names
}
