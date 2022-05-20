package completion

import (
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/spf13/cobra"
)

// ValidArgsFn a function to be used by cobra command as `ValidArgsFunction` to offer command line completion
type ValidArgsFn func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// ImageNames offers completion for images present within the local store
func ImageNames(dockerCli command.Cli) ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCli.Client().ImageList(cmd.Context(), types.ImageListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, image := range list {
			names = append(names, image.RepoTags...)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// ContainerNames offers completion for container names and IDs
// By default, only names are returned.
// Set DOCKER_COMPLETION_SHOW_CONTAINER_IDS=yes to also complete IDs.
func ContainerNames(dockerCli command.Cli, all bool, filters ...func(types.Container) bool) ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCli.Client().ContainerList(cmd.Context(), types.ContainerListOptions{
			All: all,
		})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		showContainerIDs := os.Getenv("DOCKER_COMPLETION_SHOW_CONTAINER_IDS") == "yes"

		var names []string
		for _, container := range list {
			skip := false
			for _, fn := range filters {
				if !fn(container) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			if showContainerIDs {
				names = append(names, container.ID)
			}
			names = append(names, formatter.StripNamePrefix(container.Names)...)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// VolumeNames offers completion for volumes
func VolumeNames(dockerCli command.Cli) ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCli.Client().VolumeList(cmd.Context(), filters.Args{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, volume := range list.Volumes {
			names = append(names, volume.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// NetworkNames offers completion for networks
func NetworkNames(dockerCli command.Cli) ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCli.Client().NetworkList(cmd.Context(), types.NetworkListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, network := range list {
			names = append(names, network.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// NoComplete is used for commands where there's no relevant completion
func NoComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
