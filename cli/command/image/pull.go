package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/jsonstream"
	"github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

// pullOptions defines what and how to pull.
type pullOptions struct {
	remote   string
	all      bool
	platform string
	quiet    bool
}

// newPullCommand creates a new `docker pull` command
func newPullCommand(dockerCLI command.Cli) *cobra.Command {
	var opts pullOptions

	cmd := &cobra.Command{
		Use:   "pull [OPTIONS] NAME[:TAG|@DIGEST]",
		Short: "Download an image from a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			return runPull(cmd.Context(), dockerCLI, opts)
		},
		Annotations: map[string]string{
			"category-top": "5",
			"aliases":      "docker image pull, docker pull",
		},
		// Complete with local images to help pulling the latest version
		// of images that are in the image cache.
		ValidArgsFunction:     completion.ImageNames(dockerCLI, 1),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()

	flags.BoolVarP(&opts.all, "all-tags", "a", false, "Download all tagged images in the repository")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress verbose output")

	// TODO(thaJeztah): DEPRECATED: remove in v29.1 or v30
	flags.Bool("disable-content-trust", true, "Skip image verification (deprecated)")
	_ = flags.MarkDeprecated("disable-content-trust", "support for docker content trust was removed")

	flags.StringVar(&opts.platform, "platform", os.Getenv("DOCKER_DEFAULT_PLATFORM"), "Set platform if server is multi-platform capable")
	_ = flags.SetAnnotation("platform", "version", []string{"1.32"})
	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms())

	return cmd
}

// runPull performs a pull against the engine based on the specified options
func runPull(ctx context.Context, dockerCLI command.Cli, opts pullOptions) error {
	distributionRef, err := reference.ParseNormalizedNamed(opts.remote)
	switch {
	case err != nil:
		return err
	case opts.all && !reference.IsNameOnly(distributionRef):
		return errors.New("tag can't be used with --all-tags/-a")
	case !opts.all && reference.IsNameOnly(distributionRef):
		distributionRef = reference.TagNameOnly(distributionRef)
		if tagged, ok := distributionRef.(reference.Tagged); ok && !opts.quiet {
			_, _ = fmt.Fprintln(dockerCLI.Out(), "Using default tag:", tagged.Tag())
		}
	}

	var ociPlatforms []ocispec.Platform
	if opts.platform != "" {
		// TODO(thaJeztah): add a platform option-type / flag-type.
		p, err := platforms.Parse(opts.platform)
		if err != nil {
			return err
		}
		ociPlatforms = append(ociPlatforms, p)
	}

	encodedAuth, err := command.RetrieveAuthTokenFromImage(dockerCLI.ConfigFile(), distributionRef.String())
	if err != nil {
		return err
	}

	responseBody, err := dockerCLI.Client().ImagePull(ctx, reference.FamiliarString(distributionRef), client.ImagePullOptions{
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: nil,
		All:           opts.all,
		Platforms:     ociPlatforms,
	})
	if err != nil {
		return err
	}
	defer responseBody.Close()

	out := dockerCLI.Out()
	if opts.quiet {
		out = streams.NewOut(io.Discard)
	}
	if err := jsonstream.Display(ctx, responseBody, out); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(dockerCLI.Out(), distributionRef.String())
	return nil
}
