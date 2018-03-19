package engine

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/containerizedengine"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	releaseNotePrefix = "https://docs.docker.com/releasenotes"
)

type checkOptions struct {
	registryPrefix string
	preReleases    bool
	downgrades     bool
	upgrades       bool
	format         string
	quiet          bool
	sockPath       string
}

func newCheckForUpdatesCommand(dockerCli command.Cli) *cobra.Command {
	var options checkOptions

	cmd := &cobra.Command{
		Use:   "check [OPTIONS]",
		Short: "Check for available engine updates",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(dockerCli, options)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&options.registryPrefix, "registry-prefix", "", "Override the existing location where engine images are pulled")
	flags.BoolVar(&options.downgrades, "downgrades", false, "Report downgrades (default omits older versions)")
	flags.BoolVar(&options.preReleases, "pre-releases", false, "Include pre-release versions")
	flags.BoolVar(&options.upgrades, "upgrades", true, "Report available upgrades")
	flags.StringVar(&options.format, "format", "", "Pretty-print updates using a Go template")
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display available versions")
	flags.StringVar(&options.sockPath, "containerd", "", "override default location of containerd endpoint")

	return cmd
}

func runCheck(dockerCli command.Cli, options checkOptions) error {
	ctx := context.Background()
	client, err := dockerCli.NewContainerizedEngineClient(options.sockPath)
	if err != nil {
		return errors.Wrap(err, "unable to access local containerd")
	}
	defer client.Close()
	currentOpts, err := client.GetCurrentEngineVersion(ctx)
	if err != nil {
		return err
	}

	// override with user provided prefix if specified
	if options.registryPrefix != "" {
		currentOpts.RegistryPrefix = options.registryPrefix
	}
	imageName := currentOpts.RegistryPrefix + "/" + currentOpts.EngineImage
	currentVersion := currentOpts.EngineVersion
	versions, err := client.GetEngineVersions(ctx, dockerCli.RegistryClient(false), currentVersion, imageName)
	if err != nil {
		return err
	}

	availUpdates := []containerizedengine.Update{
		{Type: "current", Version: currentVersion},
	}
	if len(versions.Patches) > 0 {
		availUpdates = append(availUpdates,
			processVersions(
				currentVersion,
				"patch",
				options.preReleases,
				versions.Patches)...)
	}
	if options.upgrades {
		availUpdates = append(availUpdates,
			processVersions(
				currentVersion,
				"upgrade",
				options.preReleases,
				versions.Upgrades)...)
	}
	if options.downgrades {
		availUpdates = append(availUpdates,
			processVersions(
				currentVersion,
				"downgrade",
				options.preReleases,
				versions.Downgrades)...)
	}

	format := options.format
	if len(format) == 0 {
		format = formatter.TableFormatKey
	}

	updatesCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewUpdatesFormat(format, options.quiet),
		Trunc:  false,
	}
	return formatter.UpdatesWrite(updatesCtx, availUpdates)
}

func processVersions(currentVersion, verType string,
	includePrerelease bool,
	versions []containerizedengine.DockerVersion) []containerizedengine.Update {
	availUpdates := []containerizedengine.Update{}
	for _, ver := range versions {
		if !includePrerelease && ver.Prerelease() != "" {
			continue
		}
		if ver.Tag != currentVersion {
			availUpdates = append(availUpdates, containerizedengine.Update{
				Type:    verType,
				Version: ver.Tag,
				Notes:   fmt.Sprintf("%s/%s", releaseNotePrefix, ver.Tag),
			})
		}
	}
	return availUpdates
}
