package image

import (
	"context"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	apiclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/spf13/cobra"
)

type tagOptions struct {
	image    string
	name     string
	platform string
}

// NewTagCommand creates a new `docker tag` command
func NewTagCommand(dockerCli command.Cli) *cobra.Command {
	var opts tagOptions

	cmd := &cobra.Command{
		Use:   "tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]",
		Short: "Create a tag TARGET_IMAGE that refers to SOURCE_IMAGE",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.image = args[0]
			opts.name = args[1]
			return runTag(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	command.AddPlatformFlag(flags, &opts.platform)

	return cmd
}

func runTag(dockerCli command.Cli, opts tagOptions) error {
	ctx := context.Background()

	err := dockerCli.Client().ImageTag(ctx, opts.image, opts.name)
	if err == nil || !apiclient.IsErrNotFound(err) {
		return err
	}

	// pull the missing image and retry tagging
	err = pullImage(ctx, dockerCli, opts.image, opts.platform, dockerCli.Err())
	if err != nil {
		return err
	}

	return dockerCli.Client().ImageTag(ctx, opts.image, opts.name)
}

func pullImage(ctx context.Context, dockerCli command.Cli, image string, platform string, out io.Writer) error {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}

	authConfig := command.ResolveAuthConfig(ctx, dockerCli, repoInfo.Index)
	encodedAuth, err := command.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}

	options := types.ImageCreateOptions{
		RegistryAuth: encodedAuth,
		Platform:     platform,
	}

	responseBody, err := dockerCli.Client().ImageCreate(ctx, image, options)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesStream(
		responseBody,
		out,
		dockerCli.Out().FD(),
		dockerCli.Out().IsTerminal(),
		nil)
}
