package image

import (
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type pullOptions struct {
	remote string
	all    bool
}

// NewPullCommand creates a new `docker pull` command
func NewPullCommand(dockerCli command.Cli) *cobra.Command {
	var opts pullOptions

	cmd := &cobra.Command{
		Use:   "pull [OPTIONS] NAME[:TAG|@DIGEST]",
		Short: "Pull an image or a repository from a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			return runPull(dockerCli, opts)
		},
	}

	flags := cmd.Flags()

	flags.BoolVarP(&opts.all, "all-tags", "a", false, "Download all tagged images in the repository")
	command.AddTrustVerificationFlags(flags)

	return cmd
}

func runPull(cli command.Cli, opts pullOptions) error {
	ctx := context.Background()
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, AuthResolver(cli), opts.remote)
	if err != nil {
		return err
	}

	distributionRef := imgRefAndAuth.Reference()
	if opts.all && !reference.IsNameOnly(distributionRef) {
		return errors.New("tag can't be used with --all-tags/-a")
	}

	if !opts.all && reference.IsNameOnly(distributionRef) {
		distributionRef = reference.TagNameOnly(distributionRef)
		if tagged, ok := distributionRef.(reference.Tagged); ok {
			fmt.Fprintf(cli.Out(), "Using default tag: %s\n", tagged.Tag())
		}
	}

	// Check if reference has a digest
	_, isCanonical := distributionRef.(reference.Canonical)
	if command.IsTrusted() && !isCanonical {
		err = trustedPull(ctx, cli, imgRefAndAuth)
	} else {
		err = imagePullPrivileged(ctx, cli, imgRefAndAuth, opts.all)
	}
	if err != nil {
		if strings.Contains(err.Error(), "when fetching 'plugin'") {
			return errors.New(err.Error() + " - Use `docker plugin install`")
		}
		return err
	}
	return nil
}
