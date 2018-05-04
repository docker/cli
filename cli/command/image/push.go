package image

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	remote    string
	untrusted bool
}

// NewPushCommand creates a new `docker push` command
func NewPushCommand(dockerCli command.Cli) *cobra.Command {
	var opts pushOptions

	cmd := &cobra.Command{
		Use:   "push [OPTIONS] NAME[:TAG]",
		Short: "Push an image or a repository to a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			return runPush(dockerCli, opts)
		},
	}

	flags := cmd.Flags()

	command.AddTrustSigningFlags(flags, &opts.untrusted, dockerCli.ContentTrustEnabled())

	return cmd
}

func runPush(dockerCli command.Cli, opts pushOptions) error {
	ref, err := reference.ParseNormalizedNamed(opts.remote)
	if err != nil {
		return err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve the Auth config relevant for this server
	authConfig := command.ResolveAuthConfig(ctx, dockerCli, repoInfo.Index)
	requestPrivilege := command.RegistryAuthenticationPrivilegedFunc(dockerCli, repoInfo.Index, "push")

	if !opts.untrusted {
		return TrustedPush(ctx, dockerCli, repoInfo, ref, authConfig, requestPrivilege)
	}

	responseBody, err := imagePushPrivileged(ctx, dockerCli, authConfig, ref, requestPrivilege)
	if err != nil {
		return err
	}

	defer responseBody.Close()
	return jsonmessage.DisplayJSONMessagesToStream(responseBody, dockerCli.Out(), nil)
}
