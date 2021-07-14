package volume

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/docker/docker/api/types"
	volumetypes "github.com/docker/docker/api/types/volume"
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
	}

	flags := cmd.Flags()
	flags.StringVar(&availability, "availability", "active", `Cluster Volume availability ("active"|"pause"|"drain")`)

	return cmd
}

func runUpdate(dockerCli command.Cli, volumeID, availability string, flags *pflag.FlagSet) error {
	// TODO(dperny): For this earliest version, the only thing that can be
	// updated is Availability, which is necessary because to delete a cluster
	// volume, the availbility must first be set to "drain"

	apiClient := dockerCli.Client()
	ctx := context.Background()

	volume, _, err := apiClient.VolumeInspectWithRaw(ctx, volumeID)
	if err != nil {
		return err
	}

	if volume.ClusterVolume == nil {
		return errors.New("Can only update cluster volumes")
	}

	if flags.Changed("availability") {
		volume.ClusterVolume.Spec.Availability = types.VolumeAvailability(availability)
	}

	return apiClient.VolumeUpdate(
		ctx, volume.ClusterVolume.ID, volume.ClusterVolume.Version,
		volumetypes.VolumeUpdateBody{
			Spec: &volume.ClusterVolume.Spec,
		},
	)
}
