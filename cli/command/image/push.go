package image

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types/image"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/moby/term"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	all       bool
	remote    string
	untrusted bool
	quiet     bool
	platform  string
}

// NewPushCommand creates a new `docker push` command
func NewPushCommand(dockerCli command.Cli) *cobra.Command {
	var opts pushOptions

	cmd := &cobra.Command{
		Use:   "push [OPTIONS] NAME[:TAG]",
		Short: "Upload an image to a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			return RunPush(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"category-top": "6",
			"aliases":      "docker image push, docker push",
		},
		ValidArgsFunction: completion.ImageNames(dockerCli),
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.all, "all-tags", "a", false, "Push all tags of an image to the repository")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress verbose output")
	command.AddTrustSigningFlags(flags, &opts.untrusted, dockerCli.ContentTrustEnabled())
	flags.StringVar(&opts.platform, "platform", os.Getenv("DOCKER_DEFAULT_PLATFORM"),
		`Push a platform-specific manifest as a single-platform image to the registry.
'os[/arch[/variant]]': Explicit platform (eg. linux/amd64)`)
	flags.SetAnnotation("platform", "version", []string{"1.46"})

	return cmd
}

// RunPush performs a push against the engine based on the specified options
func RunPush(ctx context.Context, dockerCli command.Cli, opts pushOptions) error {
	var platform *ocispec.Platform
	if opts.platform != "" {
		p, err := platforms.Parse(opts.platform)
		if err != nil {
			_, _ = fmt.Fprintf(dockerCli.Err(), "Invalid platform %s", opts.platform)
			return err
		}
		platform = &p

		printNote(dockerCli, `Selecting a single platform will only push one matching image manifest from a multi-platform image index.
This means that any other components attached to the multi-platform image index (like Buildkit attestations) won't be pushed.
If you want to only push a single platform image while preserving the attestations, please build an image with only that platform and push it instead.
Example: echo "FROM %s" | docker build - --platform %s -t <NEW-TAG>
`, opts.remote, opts.platform)
	}

	ref, err := reference.ParseNormalizedNamed(opts.remote)
	switch {
	case err != nil:
		return err
	case opts.all && !reference.IsNameOnly(ref):
		return errors.New("tag can't be used with --all-tags/-a")
	case !opts.all && reference.IsNameOnly(ref):
		ref = reference.TagNameOnly(ref)
		if tagged, ok := ref.(reference.Tagged); ok && !opts.quiet {
			_, _ = fmt.Fprintf(dockerCli.Out(), "Using default tag: %s\n", tagged.Tag())
		}
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}

	// Resolve the Auth config relevant for this server
	authConfig := command.ResolveAuthConfig(dockerCli.ConfigFile(), repoInfo.Index)
	encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
	if err != nil {
		return err
	}
	requestPrivilege := command.RegistryAuthenticationPrivilegedFunc(dockerCli, repoInfo.Index, "push")
	options := image.PushOptions{
		All:           opts.all,
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: requestPrivilege,
		Platform:      platform,
	}

	responseBody, err := dockerCli.Client().ImagePush(ctx, reference.FamiliarString(ref), options)
	if err != nil {
		return err
	}

	defer responseBody.Close()
	if !opts.untrusted {
		// TODO PushTrustedReference currently doesn't respect `--quiet`
		return PushTrustedReference(dockerCli, repoInfo, ref, authConfig, responseBody)
	}

	if opts.quiet {
		err = jsonmessage.DisplayJSONMessagesToStream(responseBody, streams.NewOut(io.Discard), nil)
		if err == nil {
			fmt.Fprintln(dockerCli.Out(), ref.String())
		}
		return err
	}
	return jsonmessage.DisplayJSONMessagesToStream(responseBody, dockerCli.Out(), nil)
}

func printNote(dockerCli command.Cli, format string, args ...any) {
	if _, isTTY := term.GetFdInfo(dockerCli.Err()); isTTY {
		_, _ = fmt.Fprint(dockerCli.Err(), "\x1b[1;37m\x1b[1;46m[ NOTE ]\x1b[0m\x1b[0m ")
	} else {
		_, _ = fmt.Fprint(dockerCli.Err(), "[ NOTE ] ")
	}
	_, _ = fmt.Fprintf(dockerCli.Err(), format+"\n\n", args...)
}
