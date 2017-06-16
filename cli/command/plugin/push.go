package plugin

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/registry"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/jsonmessage"
	dockerregistry "github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newPushCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [OPTIONS] PLUGIN[:TAG]",
		Short: "Push a plugin to a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(dockerCli, args[0])
		},
	}

	flags := cmd.Flags()

	command.AddTrustSigningFlags(flags)

	return cmd
}

func runPush(dockerCli command.Cli, name string) error {
	named, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return err
	}
	if _, ok := named.(reference.Canonical); ok {
		return errors.Errorf("invalid name: %s", name)
	}

	named = reference.TagNameOnly(named)

	ctx := context.Background()

	repoInfo, err := dockerregistry.ParseRepositoryInfo(named)
	if err != nil {
		return err
	}

	authConfig, warns, err := registry.ResolveAuthConfig(ctx, dockerCli.Client(), dockerCli.ConfigFile(), repoInfo.Index)
	for _, w := range warns {
		fmt.Fprintf(dockerCli.Err(), "Warning: %v\n", w)
	}
	if err != nil {
		return err
	}
	encodedAuth, err := registry.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}

	responseBody, err := dockerCli.Client().PluginPush(ctx, reference.FamiliarString(named), encodedAuth)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	if command.IsTrusted() {
		repoInfo.Class = "plugin"
		return image.PushTrustedReference(dockerCli, repoInfo, named, authConfig, responseBody)
	}

	return jsonmessage.DisplayJSONMessagesToStream(responseBody, dockerCli.Out(), nil)
}
