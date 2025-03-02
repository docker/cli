package volume

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	force bool

	volumes []string
}

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] VOLUME [VOLUME...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more volumes",
		Long:    "Remove one or more volumes. You cannot remove a volume that is in use by a container.",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.volumes = args
			return runRemove(cmd.Context(), dockerCli, &opts)
		},
		ValidArgsFunction: completion.VolumeNames(dockerCli),
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the removal of one or more volumes")
	flags.SetAnnotation("force", "version", []string{"1.25"})
	return cmd
}

func runRemove(ctx context.Context, dockerCLI command.Cli, opts *removeOptions) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, name := range opts.volumes {
		if err := apiClient.VolumeRemove(ctx, name, opts.force); err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}
	return errors.Join(errs...)
}
