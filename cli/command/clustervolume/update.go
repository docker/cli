package clustervolume

import (
	"context"
	"errors"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"

	"github.com/docker/docker/api/types/swarm"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newUpdateCommand(dockerCli command.Cli) *cobra.Command {
	var availability string

	cmd := &cobra.Command{
		Use:   "update VOLUME",
		Short: "Update a cluster volume",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(dockerCli, args[0], availability, cmd.Flags())
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&availability, flagAvailability, "active", `Volume availability ("active"|"pause"|"drain")`)

	return cmd
}

func runUpdate(dockerCli command.Cli, volumeID, availability string, flags *pflag.FlagSet) error {
	// Before doing any update, you must first update availability to DRAIN
	//
	// Things you can update:
	// - Labels
	// - Driver Options
	// - Secrets
	//
	// Wow that's not many things.
	//
	// For now, let's just do availability as a proof of concept.

	// if availability is not set, then let's do no update
	if !flags.Changed(flagAvailability) {
		return errors.New("must set --availability")
	}

	apiClient := dockerCli.Client()
	ctx := context.Background()

	volume, _, err := apiClient.ClusterVolumeInspectWithRaw(ctx, volumeID)
	if err != nil {
		return err
	}

	volume.Spec.Availability = swarm.VolumeAvailability(availability)

	return apiClient.ClusterVolumeUpdate(ctx, volume.ID, volume.Meta.Version, volume.Spec)
}
