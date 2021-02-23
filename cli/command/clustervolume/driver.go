package clustervolume

// TODO(dperny): This whole command tree is temporary and for the WIP testing
// preview. It will all be rippped out

import (
	"context"
	"errors"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"

	"github.com/docker/docker/api/types/swarm"

	"github.com/spf13/cobra"
)

func newAddDriverCommand(dockerCli command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "add-driver NAME CONTROLLER_SOCKET NODE_SOCKET",
		Short: "WIP TEST COMMAND: add a new CSI driver to the cluster",
		Args:  cli.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			swarmInspect, err := dockerCli.Client().SwarmInspect(context.Background())
			if err != nil {
				return err
			}

			plugin := swarm.CSIPlugin{
				Name:             args[0],
				ControllerSocket: args[1],
				NodeSocket:       args[2],
			}

			spec := swarmInspect.Spec
			spec.CSIConfig.Plugins = append(spec.CSIConfig.Plugins, plugin)

			return dockerCli.Client().SwarmUpdate(
				context.Background(), swarmInspect.Version, spec, swarm.UpdateFlags{},
			)
		},
	}
}

func newRemoveDriverCommand(dockerCli command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "rm-driver NAME",
		Short: "WIP TEST COMMAND: remove a CSI driver from the cluster",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			swarmInspect, err := dockerCli.Client().SwarmInspect(context.Background())
			if err != nil {
				return err
			}

			// don't do this, this is a horrid hack
			removeIndex := -1
			for i, plugin := range swarmInspect.Spec.CSIConfig.Plugins {
				if plugin.Name == args[0] {
					removeIndex = i
					break
				}
			}

			if removeIndex < 0 {
				return errors.New("plugin not found in list")
			}

			swarmInspect.Spec.CSIConfig.Plugins = append(
				swarmInspect.Spec.CSIConfig.Plugins[:removeIndex],
				swarmInspect.Spec.CSIConfig.Plugins[removeIndex+1:]...,
			)

			return dockerCli.Client().SwarmUpdate(
				context.Background(), swarmInspect.Version, swarmInspect.Spec, swarm.UpdateFlags{},
			)
		},
	}
}
