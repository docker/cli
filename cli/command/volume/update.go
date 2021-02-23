package volume

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newUpdateCommand(dockerCli command.Cli) *cobra.Command {
	var availability string

	cmd := &cobra.Command{
		Use:   "update [OPTIONS] [VOLUME]",
		Short: "Update a volume (cluster volumes only)",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(dockerCli, args[0], availability, cmd.Flags())
		},
		ValidArgsFunction: completion.VolumeNames(dockerCli),
	}

	flags := cmd.Flags()
	flags.StringVar(&availability, "availability", "active", `Cluster Volume availability ("active"|"pause"|"drain")`)

	return cmd
}

func runUpdate(dockerCli command.Cli, volumeID, availability string, flags *pflag.FlagSet) error {
	// TODO(dperny): For this earliest version, the only thing that can be
	// updated is Availability, which is necessary because to delete a cluster
	// volume, the availability must first be set to "drain"

	apiClient := dockerCli.Client()
	ctx := context.Background()

	vol, _, err := apiClient.VolumeInspectWithRaw(ctx, volumeID)
	if err != nil {
		return err
	}

	if vol.ClusterVolume == nil {
		return errors.New("Can only update cluster volumes")
	}

	if flags.Changed("availability") {
		vol.ClusterVolume.Spec.Availability = volume.Availability(availability)
	}

	return apiClient.VolumeUpdate(
		ctx, vol.ClusterVolume.ID, vol.ClusterVolume.Version,
		volume.UpdateOptions{
			Spec: &vol.ClusterVolume.Spec,
		},
	)
}
