package plugin

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	name      string
	untrusted bool
}

func newPushCommand(dockerCli command.Cli) *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push [OPTIONS] PLUGIN[:TAG]",
		Short: "Push a plugin to a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return runPush(dockerCli, opts)
		},
	}

	flags := cmd.Flags()

	command.AddTrustSigningFlags(flags, &opts.untrusted, dockerCli.ContentTrustEnabled())

	return cmd
}

func runPush(dockerCli command.Cli, opts pushOptions) error {
	named, err := reference.ParseNormalizedNamed(opts.name)
	if err != nil {
		return err
	}
	if _, ok := named.(reference.Canonical); ok {
		return errors.Errorf("invalid name: %s", opts.name)
	}

	named = reference.TagNameOnly(named)

	ctx := context.Background()

	repoInfo, err := registry.ParseRepositoryInfo(named)
	if err != nil {
		return err
	}
	authConfig := command.ResolveAuthConfig(ctx, dockerCli, repoInfo.Index)

	encodedAuth, err := command.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}

	responseBody, err := dockerCli.Client().PluginPush(ctx, reference.FamiliarString(named), encodedAuth)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	if !opts.untrusted {
		repoInfo.Class = "plugin"
		return image.PushTrustedReference(dockerCli, repoInfo, named, authConfig, responseBody)
	}

	return jsonmessage.DisplayJSONMessagesToStream(responseBody, dockerCli.Out(), nil)
}
