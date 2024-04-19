package image

import (
	"context"
	"fmt"
	"io"

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
	calledAs    string
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
			// Pass through how the command was invoked. We use this to print
			// warnings when an ambiguous argument was passed when using the
			// legacy (top-level) "docker images" subcommand.
			options.calledAs = cmd.CalledAs()
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
	if err := formatter.ImageWrite(imageCtx, images); err != nil {
		return err
	}
	if options.matchName != "" && len(images) == 0 && options.calledAs == "images" {
		printAmbiguousHint(dockerCLI.Err(), options.matchName)
	}
	return nil
}

// printAmbiguousHint prints an informational warning if the provided filter
// argument is ambiguous.
//
// The "docker images" top-level subcommand predates the "docker <object> <verb>"
// convention (e.g. "docker image ls"), but accepts a positional argument to
// search/filter images by name (globbing). It's common for users to accidentally
// mistake these commands, and to use (e.g.) "docker images ls", expecting
// to see all images, but ending up with an empty list because no image named
// "ls" was found.
//
// Disallowing these search-terms would be a breaking change, but we can print
// and informational message to help the users correct their mistake.
func printAmbiguousHint(stdErr io.Writer, matchName string) {
	switch matchName {
	// List of subcommands for "docker image" and their aliases (see "docker image --help"):
	case "build",
		"history",
		"import",
		"inspect",
		"list",
		"load",
		"ls",
		"prune",
		"pull",
		"push",
		"rm",
		"save",
		"tag":

		_, _ = fmt.Fprintf(stdErr, "\nNo images found matching %q: did you mean \"docker image %[1]s\"?\n", matchName)
	}
}
