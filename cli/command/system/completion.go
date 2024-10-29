package system

import (
	"strings"

	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
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
func completeEventFilters(dockerCLI completion.APIClientProvider) completion.ValidArgsFn {
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
	info, err := dockerCLI.Client().Info(cmd.Context())
	if err != nil {
		return []string{}
	}
	return []string{info.Name, info.ID}
}

// imageNames contacts the API to get a list of image names.
// In case of an error, an empty list is returned.
func imageNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	list, err := dockerCLI.Client().ImageList(cmd.Context(), image.ListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(list))
	for _, img := range list {
		names = append(names, img.RepoTags...)
	}
	return names
}

// networkNames contacts the API to get a list of network names.
// In case of an error, an empty list is returned.
func networkNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	list, err := dockerCLI.Client().NetworkList(cmd.Context(), network.ListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(list))
	for _, nw := range list {
		names = append(names, nw.Name)
	}
	return names
}

// nodeNames contacts the API to get a list of node names.
// In case of an error, an empty list is returned.
func nodeNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	list, err := dockerCLI.Client().NodeList(cmd.Context(), types.NodeListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(list))
	for _, node := range list {
		names = append(names, node.Description.Hostname)
	}
	return names
}

// volumeNames contacts the API to get a list of volume names.
// In case of an error, an empty list is returned.
func volumeNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command) []string {
	list, err := dockerCLI.Client().VolumeList(cmd.Context(), volume.ListOptions{})
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(list.Volumes))
	for _, v := range list.Volumes {
		names = append(names, v.Name)
	}
	return names
}
