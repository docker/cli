package completion

import (
	"os"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

// ValidArgsFn a function to be used by cobra command as `ValidArgsFunction` to offer command line completion
type ValidArgsFn func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// APIClientProvider provides a method to get an [client.APIClient], initializing
// it if needed.
//
// It's a smaller interface than [command.Cli], and used in situations where an
// APIClient is needed, but we want to postpone initializing the client until
// it's used.
type APIClientProvider interface {
	Client() client.APIClient
}

// ImageNames offers completion for images present within the local store
func ImageNames(dockerCLI APIClientProvider) ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCLI.Client().ImageList(cmd.Context(), image.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, img := range list {
			names = append(names, img.RepoTags...)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// ContainerNames offers completion for container names and IDs
// By default, only names are returned.
// Set DOCKER_COMPLETION_SHOW_CONTAINER_IDS=yes to also complete IDs.
func ContainerNames(dockerCLI APIClientProvider, all bool, filters ...func(types.Container) bool) ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCLI.Client().ContainerList(cmd.Context(), container.ListOptions{
			All: all,
		})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		showContainerIDs := os.Getenv("DOCKER_COMPLETION_SHOW_CONTAINER_IDS") == "yes"

		var names []string
		for _, ctr := range list {
			skip := false
			for _, fn := range filters {
				if !fn(ctr) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			if showContainerIDs {
				names = append(names, ctr.ID)
			}
			names = append(names, formatter.StripNamePrefix(ctr.Names)...)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// VolumeNames offers completion for volumes
func VolumeNames(dockerCLI APIClientProvider) ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCLI.Client().VolumeList(cmd.Context(), volume.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, vol := range list.Volumes {
			names = append(names, vol.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// NetworkNames offers completion for networks
func NetworkNames(dockerCLI APIClientProvider) ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCLI.Client().NetworkList(cmd.Context(), network.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, nw := range list {
			names = append(names, nw.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// NoComplete is used for commands where there's no relevant completion
func NoComplete(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
