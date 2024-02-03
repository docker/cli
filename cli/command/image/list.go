package image

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/image"
	"github.com/spf13/cobra"
)

type imagesOptions struct {
	matchName string

	quiet       bool
	all         bool
	noTrunc     bool
	showDigests bool
	format      string
	filter      opts.FilterOpt
}

// NewImagesCommand creates a new `docker images` command
func NewImagesCommand(dockerCLI command.Cli) *cobra.Command {
	options := imagesOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "images [OPTIONS] [REPOSITORY[:TAG]]",
		Short: "List images",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				options.matchName = args[0]
			}
			return runImages(cmd.Context(), dockerCLI, options)
		},
		Annotations: map[string]string{
			"category-top": "7",
			"aliases":      "docker image ls, docker image list, docker images",
		},
	}

	flags := cmd.Flags()

	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only show image IDs")
	flags.BoolVarP(&options.all, "all", "a", false, "Show all images (default hides intermediate images)")
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Don't truncate output")
	flags.BoolVar(&options.showDigests, "digests", false, "Show digests")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	return cmd
}

func newListCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := *NewImagesCommand(dockerCLI)
	cmd.Aliases = []string{"list"}
	cmd.Use = "ls [OPTIONS] [REPOSITORY[:TAG]]"
	return &cmd
}

func runImages(ctx context.Context, dockerCLI command.Cli, options imagesOptions) error {
	filters := options.filter.Value()
	if options.matchName != "" {
		filters.Add("reference", options.matchName)
	}

	images, err := dockerCLI.Client().ImageList(ctx, image.ListOptions{
		All:     options.all,
		Filters: filters,
	})
	if err != nil {
		return err
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCLI.ConfigFile().ImagesFormat) > 0 && !options.quiet {
			format = dockerCLI.ConfigFile().ImagesFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	imageCtx := formatter.ImageContext{
		Context: formatter.Context{
			Output: dockerCLI.Out(),
			Format: formatter.NewImageFormat(format, options.quiet, options.showDigests),
			Trunc:  !options.noTrunc,
		},
		Digest: options.showDigests,
	}
	return formatter.ImageWrite(imageCtx, images)
}
