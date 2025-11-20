// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
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
	tree        bool
}

// newImagesCommand creates a new `docker images` command
func newImagesCommand(dockerCLI command.Cli) *cobra.Command {
	options := imagesOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "images [OPTIONS] [REPOSITORY[:TAG]]",
		Short: "List images",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				options.matchName = args[0]
			}
			numImages, err := runImages(cmd.Context(), dockerCLI, options)
			if err != nil {
				return err
			}
			if numImages == 0 && options.matchName != "" && cmd.CalledAs() == "images" {
				printAmbiguousHint(dockerCLI.Err(), options.matchName)
			}
			return nil
		},
		Annotations: map[string]string{
			"category-top": "7",
			"aliases":      "docker image ls, docker image list, docker images",
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     completion.ImageNamesWithBase(dockerCLI, 1),
	}

	flags := cmd.Flags()

	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only show image IDs")
	flags.BoolVarP(&options.all, "all", "a", false, "Show all images (default hides intermediate images)")
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Don't truncate output")
	flags.BoolVar(&options.showDigests, "digests", false, "Show digests")
	flags.StringVar(&options.format, "format", "", flagsHelper.FormatHelp)
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")

	flags.BoolVar(&options.tree, "tree", false, "List multi-platform images as a tree (EXPERIMENTAL)")
	flags.SetAnnotation("tree", "version", []string{"1.47"})
	flags.SetAnnotation("tree", "experimentalCLI", nil)

	return cmd
}

func newListCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := *newImagesCommand(dockerCLI)
	cmd.Aliases = []string{"list"}
	cmd.Use = "ls [OPTIONS] [REPOSITORY[:TAG]]"
	return &cmd
}

func runImages(ctx context.Context, dockerCLI command.Cli, options imagesOptions) (int, error) {
	filters := options.filter.Value()
	if options.matchName != "" {
		filters.Add("reference", options.matchName)
	}

	useTree, err := shouldUseTree(options)
	if err != nil {
		return 0, err
	}

	listOpts := client.ImageListOptions{
		All:       options.all,
		Filters:   filters,
		Manifests: useTree,
	}

	res, err := dockerCLI.Client().ImageList(ctx, listOpts)
	if err != nil {
		return 0, err
	}

	images := res.Items
	if !options.all {
		if _, ok := filters["dangling"]; !ok {
			images = slices.DeleteFunc(images, isDangling)
		}
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCLI.ConfigFile().ImagesFormat) > 0 && !options.quiet && !options.tree {
			format = dockerCLI.ConfigFile().ImagesFormat
			useTree = false
		} else {
			format = formatter.TableFormatKey
		}
	}

	if useTree {
		return runTree(ctx, dockerCLI, treeOptions{
			images:   images,
			all:      options.all,
			filters:  filters,
			expanded: options.tree,
		})
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
		return 0, err
	}
	return len(images), nil
}

func shouldUseTree(options imagesOptions) (bool, error) {
	if options.quiet {
		if options.tree {
			return false, errors.New("--quiet is not yet supported with --tree")
		}
		return false, nil
	}
	if options.noTrunc {
		if options.tree {
			return false, errors.New("--no-trunc is not yet supported with --tree")
		}
		return false, nil
	}
	if options.showDigests {
		if options.tree {
			return false, errors.New("--show-digest is not yet supported with --tree")
		}
		return false, nil
	}
	if options.format != "" {
		if options.tree {
			return false, errors.New("--format is not yet supported with --tree")
		}
		return false, nil
	}
	return true, nil
}

// isDangling is a copy of [formatter.isDangling].
func isDangling(img image.Summary) bool {
	if len(img.RepoTags) == 0 && len(img.RepoDigests) == 0 {
		return true
	}
	return len(img.RepoTags) == 1 && img.RepoTags[0] == "<none>:<none>" && len(img.RepoDigests) == 1 && img.RepoDigests[0] == "<none>@<none>"
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
