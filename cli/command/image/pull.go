package image

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/jsonstream"
	"github.com/docker/docker/registry"
	"github.com/moby/moby/api/types/image"
	registrytypes "github.com/moby/moby/api/types/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// PullOptions defines what and how to pull
type PullOptions = pullOptions

// pullOptions defines what and how to pull.
type pullOptions struct {
	remote   string
	all      bool
	platform string
	quiet    bool
}

// NewPullCommand creates a new `docker pull` command
func NewPullCommand(dockerCli command.Cli) *cobra.Command {
	var opts pullOptions

	cmd := &cobra.Command{
		Use:   "pull [OPTIONS] NAME[:TAG|@DIGEST]",
		Short: "Download an image from a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			return runPull(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"category-top": "5",
			"aliases":      "docker image pull, docker pull",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()

	flags.BoolVarP(&opts.all, "all-tags", "a", false, "Download all tagged images in the repository")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress verbose output")

	command.AddPlatformFlag(flags, &opts.platform)

	// TODO add a (hidden) --disable-content-trust flag that throws a deprecation/removal warning and does nothing

	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)

	return cmd
}

// RunPull performs a pull against the engine based on the specified options
func RunPull(ctx context.Context, dockerCLI command.Cli, opts PullOptions) error {
	return runPull(ctx, dockerCLI, opts)
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

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(distributionRef)
	if err != nil {
		return err
	}

	authConfig := command.ResolveAuthConfig(dockerCLI.ConfigFile(), repoInfo.Index)
	encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
	if err != nil {
		return err
	}
	responseBody, err := dockerCLI.Client().ImagePull(ctx, reference.FamiliarString(distributionRef), image.PullOptions{
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: nil,
		All:           opts.all,
		Platform:      opts.platform,
	})
	if err != nil {
		if strings.Contains(err.Error(), "when fetching 'plugin'") {
			return errors.New(err.Error() + " - Use `docker plugin install`")
		}
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
