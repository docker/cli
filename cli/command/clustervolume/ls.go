package clustervolume

import (
	"context"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"

	"github.com/docker/docker/api/types"

	"github.com/fvbommel/sortorder"
	"github.com/spf13/cobra"
)

type listOptions struct{}

func newListCommand(dockerCli command.Cli) *cobra.Command {
	options := listOptions{}
	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List cluster volumes",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClusterVolumeList(dockerCli, options)
		},
	}

	return cmd
}

func runClusterVolumeList(dockerCli command.Cli, options listOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	volumes, err := client.ClusterVolumeList(ctx, types.VolumeListOptions{})
	if err != nil {
		return err
	}

	sort.Slice(volumes, func(i, j int) bool {
		return sortorder.NaturalLess(volumes[i].Spec.Name, volumes[j].Spec.Name)
	})

	volumeCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewFormat(formatter.TableFormatKey),
	}

	return FormatWrite(volumeCtx, volumes)
}
