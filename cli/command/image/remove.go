package image

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/errdefs"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	force   bool
	noPrune bool
}

// NewRemoveCommand creates a new `docker remove` command
func NewRemoveCommand(dockerCli command.Cli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:   "rmi [OPTIONS] IMAGE [IMAGE...]",
		Short: "Remove one or more images",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCli, opts, args)
		},
		ValidArgsFunction: completion.ImageNames(dockerCli, -1),
		Annotations: map[string]string{
			"aliases": "docker image rm, docker image remove, docker rmi",
		},
	}

	flags := cmd.Flags()

	flags.BoolVarP(&opts.force, "force", "f", false, "Force removal of the image")
	flags.BoolVar(&opts.noPrune, "no-prune", false, "Do not delete untagged parents")

	return cmd
}

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	cmd := *NewRemoveCommand(dockerCli)
	cmd.Aliases = []string{"rmi", "remove"}
	cmd.Use = "rm [OPTIONS] IMAGE [IMAGE...]"
	return &cmd
}

func runRemove(ctx context.Context, dockerCLI command.Cli, opts removeOptions, images []string) error {
	apiClient := dockerCLI.Client()

	options := image.RemoveOptions{
		Force:         opts.force,
		PruneChildren: !opts.noPrune,
	}

	// TODO(thaJeztah): this logic can likely be simplified: do we want to print "not found" errors at all when using "force"?
	fatalErr := false
	var errs []error
	for _, img := range images {
		dels, err := apiClient.ImageRemove(ctx, img, options)
		if err != nil {
			if !errdefs.IsNotFound(err) {
				fatalErr = true
			}
			errs = append(errs, err)
		} else {
			for _, del := range dels {
				if del.Deleted != "" {
					_, _ = fmt.Fprintln(dockerCLI.Out(), "Deleted:", del.Deleted)
				} else {
					_, _ = fmt.Fprintln(dockerCLI.Out(), "Untagged:", del.Untagged)
				}
			}
		}
	}

	if err := errors.Join(errs...); err != nil {
		if !opts.force || fatalErr {
			return err
		}
		_, _ = fmt.Fprintln(dockerCLI.Err(), err)
	}
	return nil
}
