package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/errdefs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type tagOptions struct {
	image string
	names []string
}

// NewTagCommand creates a new `docker tag` command
func NewTagCommand(dockerCli command.Cli) *cobra.Command {
	var opts tagOptions

	cmd := &cobra.Command{
		Use:   "tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG] [TARGET_IMAGE[:TAG]...]",
		Short: "Create one or more tags TARGET_IMAGE that refers to SOURCE_IMAGE",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.image = args[0]
			opts.names = args[1:]
			return runTag(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"aliases": "docker image tag, docker tag",
		},
		ValidArgsFunction: completion.ImageNames(dockerCli),
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	return cmd
}

func runTag(ctx context.Context, dockerCli command.Cli, opts tagOptions) error {
	var errs []string
	fatalErr := false
	for _, name := range opts.names {
		err := dockerCli.Client().ImageTag(ctx, opts.image, name)
		if err != nil {
			if !errdefs.IsNotFound(err) {
				fatalErr = true
			}
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		msg := strings.Join(errs, "\n")
		if fatalErr {
			return errors.New(msg)
		}
		fmt.Fprintln(dockerCli.Err(), msg)
	}
	return nil
}
