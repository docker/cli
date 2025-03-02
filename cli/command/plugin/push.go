package plugin

import (
	"context"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/internal/jsonstream"
	registrytypes "github.com/docker/docker/api/types/registry"
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
			return runPush(cmd.Context(), dockerCli, opts)
		},
	}

	flags := cmd.Flags()

	command.AddTrustSigningFlags(flags, &opts.untrusted, dockerCli.ContentTrustEnabled())

	return cmd
}

func runPush(ctx context.Context, dockerCli command.Cli, opts pushOptions) error {
	named, err := reference.ParseNormalizedNamed(opts.name)
	if err != nil {
		return err
	}
	if _, ok := named.(reference.Canonical); ok {
		return errors.Errorf("invalid name: %s", opts.name)
	}

	named = reference.TagNameOnly(named)

	repoInfo, err := registry.ParseRepositoryInfo(named)
	if err != nil {
		return err
	}
	authConfig := command.ResolveAuthConfig(dockerCli.ConfigFile(), repoInfo.Index)
	encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
	if err != nil {
		return err
	}

	responseBody, err := dockerCli.Client().PluginPush(ctx, reference.FamiliarString(named), encodedAuth)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	if !opts.untrusted {
		return image.PushTrustedReference(ctx, dockerCli, repoInfo, named, authConfig, responseBody)
	}

	return jsonstream.Display(ctx, responseBody, dockerCli.Out())
}
