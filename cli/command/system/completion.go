package system

import (
	"strings"

	"github.com/docker/docker/api/types/events"

	"github.com/docker/cli/cli/command/completion"
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
			return prefixWith("event=", validEventNames()), cobra.ShellCompDirectiveDefault
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
			return prefixWith("type=", eventTypeNames()), cobra.ShellCompDirectiveDefault
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
	names := []string{}
	for _, eventAction := range eventActions {
		if strings.Contains(string(eventAction), " ") {
			continue
		}
		names = append(names, string(eventAction))
	}
	return names
}
