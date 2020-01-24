package image

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	remote    string
	untrusted bool
	quiet     bool
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
			return RunPush(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress verbose output")
	command.AddTrustSigningFlags(flags, &opts.untrusted, dockerCli.ContentTrustEnabled())

	return cmd
}

// RunPush performs a push against the engine based on the specified options
func RunPush(dockerCli command.Cli, opts pushOptions) error {
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
	if opts.quiet {
		err = jsonmessage.DisplayJSONMessagesToStream(responseBody, streams.NewOut(ioutil.Discard), nil)
		if err == nil {
			fmt.Fprintln(dockerCli.Out(), ref.String())
		}
		return err
	}
	return jsonmessage.DisplayJSONMessagesToStream(responseBody, dockerCli.Out(), nil)
}
