package engine

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/versions"
	clitypes "github.com/docker/cli/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type checkOptions struct {
	registryPrefix string
	preReleases    bool
	engineImage    string
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
	flags.StringVar(&options.registryPrefix, "registry-prefix", clitypes.RegistryPrefix, "Override the existing location where engine images are pulled")
	flags.BoolVar(&options.downgrades, "downgrades", false, "Report downgrades (default omits older versions)")
	flags.BoolVar(&options.preReleases, "pre-releases", false, "Include pre-release versions")
	flags.StringVar(&options.engineImage, "engine-image", "", "Specify engine image (default uses the same image as currently running)")
	flags.BoolVar(&options.upgrades, "upgrades", true, "Report available upgrades")
	flags.StringVar(&options.format, "format", "", "Pretty-print updates using a Go template")
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display available versions")
	flags.StringVar(&options.sockPath, "containerd", "", "override default location of containerd endpoint")

	return cmd
}

func runCheck(dockerCli command.Cli, options checkOptions) error {
	if !isRoot() {
		return errors.New("this command must be run as a privileged user")
	}
	ctx := context.Background()
	client := dockerCli.Client()
	serverVersion, err := client.ServerVersion(ctx)
	if err != nil {
		return err
	}

	availVersions, err := versions.GetEngineVersions(ctx, dockerCli.RegistryClient(false), options.registryPrefix, options.engineImage, serverVersion.Version)
	if err != nil {
		return err
	}

	availUpdates := []clitypes.Update{
		{Type: "current", Version: serverVersion.Version},
	}
	if len(availVersions.Patches) > 0 {
		availUpdates = append(availUpdates,
			processVersions(
				serverVersion.Version,
				"patch",
				options.preReleases,
				availVersions.Patches)...)
	}
	if options.upgrades {
		availUpdates = append(availUpdates,
			processVersions(
				serverVersion.Version,
				"upgrade",
				options.preReleases,
				availVersions.Upgrades)...)
	}
	if options.downgrades {
		availUpdates = append(availUpdates,
			processVersions(
				serverVersion.Version,
				"downgrade",
				options.preReleases,
				availVersions.Downgrades)...)
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
	availVersions []clitypes.DockerVersion) []clitypes.Update {
	availUpdates := []clitypes.Update{}
	for _, ver := range availVersions {
		if !includePrerelease && ver.Prerelease() != "" {
			continue
		}
		if ver.Tag != currentVersion {
			availUpdates = append(availUpdates, clitypes.Update{
				Type:    verType,
				Version: ver.Tag,
				Notes:   fmt.Sprintf("%s/%s", clitypes.ReleaseNotePrefix, ver.Tag),
			})
		}
	}
	return availUpdates
}
