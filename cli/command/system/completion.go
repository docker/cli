package system

import (
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

var (
	eventFilters = []string{"container", "daemon", "event", "image", "label", "network", "node", "scope", "type", "volume"}

	// eventTypes is a list of all event types.
	// This should be moved to the moby codebase once its usage is consolidated here.
	eventTypes = []events.Type{
		events.BuilderEventType,
		events.ConfigEventType,
		events.ContainerEventType,
		events.DaemonEventType,
		events.ImageEventType,
		events.NetworkEventType,
		events.NodeEventType,
		events.PluginEventType,
		events.SecretEventType,
		events.ServiceEventType,
		events.VolumeEventType,
	}

	// eventActions is a list of all event actions.
	// This should be moved to the moby codebase once its usage is consolidated here.
	eventActions = []events.Action{
		events.ActionCreate,
		events.ActionStart,
		events.ActionRestart,
		events.ActionStop,
		events.ActionCheckpoint,
		events.ActionPause,
		events.ActionUnPause,
		events.ActionAttach,
		events.ActionDetach,
		events.ActionResize,
		events.ActionUpdate,
		events.ActionRename,
		events.ActionKill,
		events.ActionDie,
		events.ActionOOM,
		events.ActionDestroy,
		events.ActionRemove,
		events.ActionCommit,
		events.ActionTop,
		events.ActionCopy,
		events.ActionArchivePath,
		events.ActionExtractToDir,
		events.ActionExport,
		events.ActionImport,
		events.ActionSave,
		events.ActionLoad,
		events.ActionTag,
		events.ActionUnTag,
		events.ActionPush,
		events.ActionPull,
		events.ActionPrune,
		events.ActionDelete,
		events.ActionEnable,
		events.ActionDisable,
		events.ActionConnect,
		events.ActionDisconnect,
		events.ActionReload,
		events.ActionMount,
		events.ActionUnmount,
		events.ActionExecCreate,
		events.ActionExecStart,
		events.ActionExecDie,
		events.ActionExecDetach,
		events.ActionHealthStatus,
		events.ActionHealthStatusRunning,
		events.ActionHealthStatusHealthy,
		events.ActionHealthStatusUnhealthy,
	}
)

// completeEventFilters provides completion for the filters that can be used with `--filter`.
func completeEventFilters(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		key, _, ok := strings.Cut(toComplete, "=")
		if !ok {
			return postfixWith("=", eventFilters), cobra.ShellCompDirectiveNoSpace
		}
		switch key {
		case "container":
			return prefixWith("container=", containerNames(dockerCLI, cmd, args, toComplete)), cobra.ShellCompDirectiveNoFileComp
		case "daemon":
			return prefixWith("daemon=", daemonNames(dockerCLI, cmd)), cobra.ShellCompDirectiveNoFileComp
		case "event":
			return prefixWith("event=", validEventNames()), cobra.ShellCompDirectiveNoFileComp
		case "image":
			return prefixWith("image=", imageNames(dockerCLI, cmd)), cobra.ShellCompDirectiveNoFileComp
		case "label":
			return nil, cobra.ShellCompDirectiveNoFileComp
		case "network":
			return prefixWith("network=", networkNames(dockerCLI, cmd)), cobra.ShellCompDirectiveNoFileComp
		case "node":
			return prefixWith("node=", nodeNames(dockerCLI, cmd)), cobra.ShellCompDirectiveNoFileComp
		case "scope":
			return prefixWith("scope=", []string{"local", "swarm"}), cobra.ShellCompDirectiveNoFileComp
		case "type":
			return prefixWith("type=", eventTypeNames()), cobra.ShellCompDirectiveNoFileComp
		case "volume":
			return prefixWith("volume=", volumeNames(dockerCLI, cmd)), cobra.ShellCompDirectiveNoFileComp
		default:
			return postfixWith("=", eventFilters), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
	}
}

// prefixWith prefixes every element in the slice with the given prefix.
func prefixWith(prefix string, values []string) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = prefix + v
	}
	return result
}

// postfixWith appends postfix to every element in the slice.
func postfixWith(postfix string, values []string) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v + postfix
	}
	return result
}

// eventTypeNames provides a list of all event types.
// The list is derived from eventTypes.
func eventTypeNames() []string {
	names := make([]string, len(eventTypes))
	for i, eventType := range eventTypes {
		names[i] = string(eventType)
	}
	return names
}

// validEventNames provides a list of all event actions.
// The list is derived from eventActions.
// Actions that are not suitable for usage in completions are removed.
func validEventNames() []string {
	names := make([]string, 0, len(eventActions))
	for _, eventAction := range eventActions {
		if strings.Contains(string(eventAction), " ") {
			continue
		}
		names = append(names, string(eventAction))
	}
	return names
}

// configNames contacts the API to get a list of config names.
// In case of an error, an empty list is returned.
func configNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().ConfigList(cmd.Context(), client.ConfigListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(res.Items))
	for _, v := range res.Items {
		names = append(names, v.Spec.Name)
	}
	return names
}

// containerNames contacts the API to get names and optionally IDs of containers.
// In case of an error, an empty list is returned.
func containerNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command, args []string, toComplete string) []string {
	names, _ := completion.ContainerNames(dockerCLI, true)(cmd, args, toComplete)
	if names == nil {
		return []string{}
	}
	return names
}

// daemonNames contacts the API to get name and ID of the current docker daemon.
// In case of an error, an empty list is returned.
func daemonNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().Info(cmd.Context(), client.InfoOptions{})
	if err != nil {
		return []string{}
	}
	return []string{res.Info.Name, res.Info.ID}
}

// imageNames contacts the API to get a list of image names.
// In case of an error, an empty list is returned.
func imageNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().ImageList(cmd.Context(), client.ImageListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(res.Items))
	for _, img := range res.Items {
		names = append(names, img.RepoTags...)
	}
	return names
}

// networkNames contacts the API to get a list of network names.
// In case of an error, an empty list is returned.
func networkNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().NetworkList(cmd.Context(), client.NetworkListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(res.Items))
	for _, nw := range res.Items {
		names = append(names, nw.Name)
	}
	return names
}

// nodeNames contacts the API to get a list of node names.
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

// pluginNames contacts the API to get a list of plugin names.
// In case of an error, an empty list is returned.
func pluginNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().PluginList(cmd.Context(), client.PluginListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(res.Items))
	for _, v := range res.Items {
		names = append(names, v.Name)
	}
	return names
}

// secretNames contacts the API to get a list of secret names.
// In case of an error, an empty list is returned.
func secretNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().SecretList(cmd.Context(), client.SecretListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(res.Items))
	for _, v := range res.Items {
		names = append(names, v.Spec.Name)
	}
	return names
}

// serviceNames contacts the API to get a list of service names.
// In case of an error, an empty list is returned.
func serviceNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().ServiceList(cmd.Context(), client.ServiceListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(res.Items))
	for _, v := range res.Items {
		names = append(names, v.Spec.Name)
	}
	return names
}

// taskNames contacts the API to get a list of service names.
// In case of an error, an empty list is returned.
func taskNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().TaskList(cmd.Context(), client.TaskListOptions{})
	if err != nil || len(res.Items) == 0 {
		return []string{}
	}

	resolver := idresolver.New(dockerCLI.Client(), false)
	names := make([]string, 0, len(res.Items))
	for _, task := range res.Items {
		serviceName, err := resolver.Resolve(cmd.Context(), swarm.Service{}, task.ServiceID)
		if err != nil {
			continue
		}
		if task.Slot != 0 {
			names = append(names, fmt.Sprintf("%v.%v", serviceName, task.Slot))
		} else {
			names = append(names, fmt.Sprintf("%v.%v", serviceName, task.NodeID))
		}
	}
	return names
}

// volumeNames contacts the API to get a list of volume names.
// In case of an error, an empty list is returned.
func volumeNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	res, err := dockerCLI.Client().VolumeList(cmd.Context(), client.VolumeListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(res.Items))
	for _, v := range res.Items {
		names = append(names, v.Name)
	}
	return names
}

// completeObjectNames completes names of objects based on the "--type" flag
//
// TODO(thaJeztah): completion functions in this package don't remove names that have already been completed
// this causes completion to continue even if a given name was already completed.
func completeObjectNames(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if f := cmd.Flags().Lookup("type"); f != nil && f.Changed {
			switch f.Value.String() {
			case typeConfig:
				return configNames(dockerCLI, cmd), cobra.ShellCompDirectiveNoFileComp
			case typeContainer:
				return containerNames(dockerCLI, cmd, args, toComplete), cobra.ShellCompDirectiveNoFileComp
			case typeImage:
				return imageNames(dockerCLI, cmd), cobra.ShellCompDirectiveNoFileComp
			case typeNetwork:
				return networkNames(dockerCLI, cmd), cobra.ShellCompDirectiveNoFileComp
			case typeNode:
				return nodeNames(dockerCLI, cmd), cobra.ShellCompDirectiveNoFileComp
			case typePlugin:
				return pluginNames(dockerCLI, cmd), cobra.ShellCompDirectiveNoFileComp
			case typeSecret:
				return secretNames(dockerCLI, cmd), cobra.ShellCompDirectiveNoFileComp
			case typeService:
				return serviceNames(dockerCLI, cmd), cobra.ShellCompDirectiveNoFileComp
			case typeTask:
				return taskNames(dockerCLI, cmd), cobra.ShellCompDirectiveNoFileComp
			case typeVolume:
				return volumeNames(dockerCLI, cmd), cobra.ShellCompDirectiveNoFileComp
			default:
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
