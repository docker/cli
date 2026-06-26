package volume

import (
	"context"
	"errors"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newUpdateCommand(dockerCLI command.Cli) *cobra.Command {
	var availability string

	cmd := &cobra.Command{
		Use:   "update [OPTIONS] [VOLUME]",
		Short: "Update a volume (cluster volumes only)",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), dockerCLI, args[0], availability, cmd.Flags())
		},
		Annotations: map[string]string{
			"version": "1.42",
			"swarm":   "manager",
		},
		ValidArgsFunction:     completion.VolumeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&availability, "availability", "active", `Cluster Volume availability ("active", "pause", "drain")`)
	_ = flags.SetAnnotation("availability", "version", []string{"1.42"})
	_ = flags.SetAnnotation("availability", "swarm", []string{"manager"})

	return cmd
}

func runUpdate(ctx context.Context, dockerCli command.Cli, volumeID, availability string, flags *pflag.FlagSet) error {
	// TODO(dperny): For this earliest version, the only thing that can be
	// updated is Availability, which is necessary because to delete a cluster
	// volume, the availability must first be set to "drain"

	apiClient := dockerCli.Client()

	res, err := apiClient.VolumeInspect(ctx, volumeID, client.VolumeInspectOptions{})
	if err != nil {
		return err
	}

	if res.Volume.ClusterVolume == nil {
		return errors.New("can only update cluster volumes")
	}

	if flags.Changed("availability") {
		res.Volume.ClusterVolume.Spec.Availability = volume.Availability(availability)
	}
	_, err = apiClient.VolumeUpdate(ctx, res.Volume.ClusterVolume.ID, client.VolumeUpdateOptions{
		Version: res.Volume.ClusterVolume.Version,
		Spec:    &res.Volume.ClusterVolume.Spec,
	})
	return err
}
