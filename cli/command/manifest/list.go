package manifest

import (
	"sort"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/cli/manifest/types"
	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
)

type listOptions struct {
	quiet  bool
	format string
}

func newListCommand(dockerCli command.Cli) *cobra.Command {
	var options listOptions

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List local manifest lists",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(dockerCli, options)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only show manifest list NAMEs")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	return cmd
}

func runList(dockerCli command.Cli, options listOptions) error {
	manifestStore := dockerCli.ManifestStore()

	var manifestLists []reference.Reference

	manifestLists, err := manifestStore.List()
	if err != nil {
		return err
	}

	manifests := map[string][]types.ImageManifest{}
	for _, manifestList := range manifestLists {
		if imageManifests, err := manifestStore.GetList(manifestList); err == nil {
			manifests[manifestList.String()] = imageManifests
		}
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().ManifestListsFormat) > 0 && !options.quiet {
			format = dockerCli.ConfigFile().ManifestListsFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	manifestListsCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(format, options.quiet),
	}
	sort.Slice(manifestLists, func(i, j int) bool {
		return sortorder.NaturalLess(manifestLists[i].String(), manifestLists[j].String())
	})
	return FormatWrite(manifestListsCtx, manifestLists, manifests)
}
