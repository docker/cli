package plugin

import (
	"context"
	"fmt"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/jsonstream"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

func newPushCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [OPTIONS] PLUGIN[:TAG]",
		Short: "Push a plugin to a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return runPush(cmd.Context(), dockerCLI, name)
		},
		ValidArgsFunction:     completeNames(dockerCLI, stateAny),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	// TODO(thaJeztah): DEPRECATED: remove in v29.1 or v30
	flags.Bool("disable-content-trust", true, "Skip image verification (deprecated)")
	_ = flags.MarkDeprecated("disable-content-trust", "support for docker content trust was removed")
	return cmd
}

func runPush(ctx context.Context, dockerCli command.Cli, name string) error {
	named, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return err
	}
	if _, ok := named.(reference.Canonical); ok {
		return fmt.Errorf("invalid name: %s", name)
	}

	named = reference.TagNameOnly(named)
	encodedAuth, err := command.RetrieveAuthTokenFromImage(dockerCli.ConfigFile(), named.String())
	if err != nil {
		return err
	}

	responseBody, err := dockerCli.Client().PluginPush(ctx, reference.FamiliarString(named), client.PluginPushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = responseBody.Close()
	}()
	return jsonstream.Display(ctx, responseBody, dockerCli.Out())
}
