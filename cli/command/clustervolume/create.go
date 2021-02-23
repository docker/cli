package clustervolume

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
)

func newCreateCommand(dockerCli command.Cli) *cobra.Command {
	opts := newClusterVolumeOptions()

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] NAME",
		Short: "Create a new cluster volume",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return runCreate(dockerCli, cmd.Flags(), opts)
		},
	}

	flags := cmd.Flags()
	addVolumeFlags(flags, opts)

	return cmd
}

func runCreate(dockerCli command.Cli, flags *pflag.FlagSet, opts *clusterVolumeOptions) error {
	apiClient := dockerCli.Client()
	ctx := context.Background()

	volumeSpec := opts.ToVolumeSpec()

	response, err := apiClient.ClusterVolumeCreate(ctx, volumeSpec)
	if err != nil {
		return err
	}

	fmt.Fprintf(dockerCli.Out(), "%s\n", response.ID)
	return nil
}
