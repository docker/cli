package volume

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	force bool

	volumes []string
}

func newRemoveCommand(dockerCLI command.Cli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] VOLUME [VOLUME...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more volumes",
		Long:    "Remove one or more volumes. You cannot remove a volume that is in use by a container.",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.volumes = args
			return runRemove(cmd.Context(), dockerCLI, &opts)
		},
		ValidArgsFunction:     completion.VolumeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the removal of one or more volumes")
	_ = flags.SetAnnotation("force", "version", []string{"1.25"})
	return cmd
}

func runRemove(ctx context.Context, dockerCLI command.Cli, opts *removeOptions) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, name := range opts.volumes {
		_, err := apiClient.VolumeRemove(ctx, name, client.VolumeRemoveOptions{
			Force: opts.force,
		})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}
	return errors.Join(errs...)
}
