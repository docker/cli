package completion

import (
	"os"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// APIClientProvider provides a method to get a [client.APIClient], initializing
// it if needed.
//
// It's a smaller interface than [command.Cli], and used in situations where an
// APIClient is needed, but we want to postpone initializing the client until
// it's used.
type APIClientProvider interface {
	Client() client.APIClient
}

// ImageNames offers completion for images present within the local store
func ImageNames(dockerCLI APIClientProvider, limit int) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if limit > 0 && len(args) >= limit {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		res, err := dockerCLI.Client().ImageList(cmd.Context(), client.ImageListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, img := range res.Items {
			names = append(names, img.RepoTags...)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// ImageNamesWithBase offers completion for images present within the local store,
// including both full image names with tags and base image names (repository names only)
// when multiple tags exist for the same base name
func ImageNamesWithBase(dockerCLI APIClientProvider, limit int) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if limit > 0 && len(args) >= limit {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		res, err := dockerCLI.Client().ImageList(cmd.Context(), client.ImageListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		baseNameCounts := make(map[string]int)
		for _, img := range res.Items {
			names = append(names, img.RepoTags...)
			for _, tag := range img.RepoTags {
				ref, err := reference.ParseNormalizedNamed(tag)
				if err != nil {
					continue
				}
				baseNameCounts[reference.FamiliarName(ref)]++
			}
		}
		for baseName, count := range baseNameCounts {
			if count > 1 {
				names = append(names, baseName)
			}
		}
		return names, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}
}

// ContainerNames offers completion for container names and IDs
// By default, only names are returned.
// Set DOCKER_COMPLETION_SHOW_CONTAINER_IDS=yes to also complete IDs.
func ContainerNames(dockerCLI APIClientProvider, all bool, filters ...func(container.Summary) bool) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		res, err := dockerCLI.Client().ContainerList(cmd.Context(), client.ContainerListOptions{
			All: all,
		})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		showContainerIDs := os.Getenv("DOCKER_COMPLETION_SHOW_CONTAINER_IDS") == "yes"

		var names []string
		for _, ctr := range res.Items {
			skip := false
			for _, fn := range filters {
				if fn != nil && !fn(ctr) {
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
func VolumeNames(dockerCLI APIClientProvider) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		res, err := dockerCLI.Client().VolumeList(cmd.Context(), client.VolumeListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, vol := range res.Items {
			names = append(names, vol.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// NetworkNames offers completion for networks
func NetworkNames(dockerCLI APIClientProvider) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		res, err := dockerCLI.Client().NetworkList(cmd.Context(), client.NetworkListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, nw := range res.Items {
			names = append(names, nw.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// EnvVarNames offers completion for environment-variable names. This
// completion can be used for "--env" and "--build-arg" flags, which
// allow obtaining the value of the given environment-variable if present
// in the local environment, so we only should complete the names of the
// environment variables, and not their value. This also prevents the
// completion script from printing values of environment variables
// containing sensitive values.
//
// For example;
//
//	export MY_VAR=hello
//	docker run --rm --env MY_VAR alpine printenv MY_VAR
//	hello
func EnvVarNames() cobra.CompletionFunc {
	return func(_ *cobra.Command, _ []string, _ string) (names []string, _ cobra.ShellCompDirective) {
		envs := os.Environ()
		names = make([]string, 0, len(envs))
		for _, env := range envs {
			name, _, _ := strings.Cut(env, "=")
			names = append(names, name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// FromList offers completion for the given list of options.
func FromList(options ...string) cobra.CompletionFunc {
	return cobra.FixedCompletions(options, cobra.ShellCompDirectiveNoFileComp)
}

// FileNames is a convenience function to use [cobra.ShellCompDirectiveDefault],
// which indicates to let the shell perform its default behavior after
// completions have been provided.
func FileNames() cobra.CompletionFunc {
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	}
}

var commonPlatforms = []string{
	"linux/386",
	"linux/amd64",
	"linux/arm",
	"linux/arm/v5",
	"linux/arm/v6",
	"linux/arm/v7",
	"linux/arm64",
	"linux/arm64/v8",

	// IBM power and z platforms
	"linux/ppc64le",
	"linux/s390x",

	// Not yet supported
	"linux/riscv64",

	"windows/amd64",

	"wasip1/wasm",
}

// Platforms offers completion for platform-strings. It provides a non-exhaustive
// list of platforms to be used for completion. Platform-strings are based on
// [runtime.GOOS] and [runtime.GOARCH], but with (optional) variants added. A
// list of recognised os/arch combinations from the Go runtime can be obtained
// through "go tool dist list".
//
// Some noteworthy exclusions from this list:
//
//   - arm64 images ("windows/arm64", "windows/arm64/v8") do not yet exist for windows.
//   - we don't (yet) include `os-variant` for completion (as can be used for Windows images)
//   - we don't (yet) include platforms for which we don't build binaries, such as
//     BSD platforms (freebsd, netbsd, openbsd), android, macOS (darwin).
//   - we currently exclude architectures that may have unofficial builds,
//     but don't have wide adoption (and no support), such as loong64, mipsXXX,
//     ppc64 (non-le) to prevent confusion.
func Platforms() cobra.CompletionFunc {
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return commonPlatforms, cobra.ShellCompDirectiveNoFileComp
	}
}
