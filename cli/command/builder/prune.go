package builder

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

// NewPruneCommand returns a new cobra prune command for images
func NewPruneCommand(dockerCli command.Cli) *cobra.Command {
	options := types.BuildCachePruneOptions{}

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove build cache",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := dockerCli.Client().BuildCachePrune(context.Background(), options)
			if err != nil {
				return err
			}
			fmt.Fprintln(dockerCli.Out(), "Total reclaimed space:", units.HumanSize(float64(report.SpaceReclaimed)))
			return nil
		},
		Annotations: map[string]string{"version": "1.39"},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.All, "all", "a", false, "Remove all build cache, including internal/frontend references")
	flags.DurationVar(&options.KeepDuration, "keep-duration", 0, "Keep build cache data newer than <duration> ago (default: 0)")
	flags.Float64Var(&options.KeepStorage, "keep-storage", 0, "Keep total build cache size below this limit (in MB) (default: 0)")

	return cmd
}
