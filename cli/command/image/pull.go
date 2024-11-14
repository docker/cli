package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/docker/api/types/hub"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// PullOptions defines what and how to pull
type PullOptions struct {
	remote    string
	all       bool
	platform  string
	quiet     bool
	untrusted bool
}

// NewPullCommand creates a new `docker pull` command
func NewPullCommand(dockerCli command.Cli) *cobra.Command {
	var opts PullOptions

	cmd := &cobra.Command{
		Use:   "pull [OPTIONS] NAME[:TAG|@DIGEST]",
		Short: "Download an image from a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			return RunPull(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"category-top": "5",
			"aliases":      "docker image pull, docker pull",
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			if image, tag, found := strings.Cut(toComplete, ":"); found {
				remoteTags, err := dockerCli.Client().HubImageTags(cmd.Context(), image, hub.ImageOptions{
					Name:     tag,
					Ordering: "last_updated",
					Page:     0,
					PageSize: 25,
				})
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}

				images := make([]string, 0, len(remoteTags.Results))
				for _, tag := range remoteTags.Results {
					fullName := image + ":" + tag.Name
					images = append(images, fullName+"\t"+tag.LastUpdated.String())
				}
				return images, cobra.ShellCompDirectiveKeepOrder | cobra.ShellCompDirectiveNoFileComp
			}

			remoteImages, err := dockerCli.Client().HubImageSearch(cmd.Context(), toComplete, hub.SearchOptions{
				From:              0,
				Size:              25,
				Type:              hub.SearchTypeImage,
				Order:             hub.SearchOrderDesc,
				Official:          true,
				Source:            hub.SearchSourceStore,
				OpenSource:        true,
				ExtensionReviewed: true,
			})
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			images := make([]string, 0, len(remoteImages.Results))
			for _, img := range remoteImages.Results {
				categories := make([]string, 0, len(img.Categories))
				for _, cat := range img.Categories {
					categories = append(categories, cat.Name)
				}
				images = append(images, img.Name+"\t"+strings.Join(categories, ", "))
			}

			return images, cobra.ShellCompDirectiveKeepOrder | cobra.ShellCompDirectiveNoFileComp
		},
	}

	flags := cmd.Flags()

	flags.BoolVarP(&opts.all, "all-tags", "a", false, "Download all tagged images in the repository")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress verbose output")

	command.AddPlatformFlag(flags, &opts.platform)
	command.AddTrustVerificationFlags(flags, &opts.untrusted, dockerCli.ContentTrustEnabled())

	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)

	return cmd
}

// RunPull performs a pull against the engine based on the specified options
func RunPull(ctx context.Context, dockerCLI command.Cli, opts PullOptions) error {
	distributionRef, err := reference.ParseNormalizedNamed(opts.remote)
	switch {
	case err != nil:
		return err
	case opts.all && !reference.IsNameOnly(distributionRef):
		return errors.New("tag can't be used with --all-tags/-a")
	case !opts.all && reference.IsNameOnly(distributionRef):
		distributionRef = reference.TagNameOnly(distributionRef)
		if tagged, ok := distributionRef.(reference.Tagged); ok && !opts.quiet {
			fmt.Fprintf(dockerCLI.Out(), "Using default tag: %s\n", tagged.Tag())
		}
	}

	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, AuthResolver(dockerCLI), distributionRef.String())
	if err != nil {
		return err
	}

	// Check if reference has a digest
	_, isCanonical := distributionRef.(reference.Canonical)
	if !opts.untrusted && !isCanonical {
		err = trustedPull(ctx, dockerCLI, imgRefAndAuth, opts)
	} else {
		err = imagePullPrivileged(ctx, dockerCLI, imgRefAndAuth, opts)
	}
	if err != nil {
		if strings.Contains(err.Error(), "when fetching 'plugin'") {
			return errors.New(err.Error() + " - Use `docker plugin install`")
		}
		return err
	}
	fmt.Fprintln(dockerCLI.Out(), imgRefAndAuth.Reference().String())
	return nil
}
