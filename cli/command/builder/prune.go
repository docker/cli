package builder

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

// NewPruneCommand returns a new cobra prune command for images
func NewPruneCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove build cache",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := dockerCli.Client().BuildCachePrune(context.Background())
			if err != nil {
				return err
			}
			fmt.Fprintln(dockerCli.Out(), "Total reclaimed space:", units.HumanSize(float64(report.SpaceReclaimed)))
			return nil
		},
		Annotations: map[string]string{"version": "1.39"},
	}

	return cmd
}
