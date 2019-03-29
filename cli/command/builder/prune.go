package builder

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

type pruneOptions struct {
	force       bool
	all         bool
	filter      opts.FilterOpt
	keepStorage opts.MemBytes
}

// NewPruneCommand returns a new cobra prune command for images
func NewPruneCommand(dockerCli command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove build cache",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceReclaimed, output, err := runPrune(dockerCli, options)
			if err != nil {
				return err
			}
			if output != "" {
				fmt.Fprintln(dockerCli.Out(), output)
			}
			fmt.Fprintln(dockerCli.Out(), "Total reclaimed space:", units.HumanSize(float64(spaceReclaimed)))
			return nil
		},
		Annotations: map[string]string{"version": "1.39"},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.force, "force", "f", false, "Do not prompt for confirmation")
	flags.BoolVarP(&options.all, "all", "a", false, "Remove all unused images, not just dangling ones")
	flags.Var(&options.filter, "filter", "Provide filter values (e.g. 'unused-for=24h')")
	flags.Var(&options.keepStorage, "keep-storage", "Amount of disk space to keep for cache")

	return cmd
}

const (
	normalWarning   = `WARNING! This will remove all dangling build cache. Are you sure you want to continue?`
	allCacheWarning = `WARNING! This will remove all build cache. Are you sure you want to continue?`
)

func runPrune(dockerCli command.Cli, options pruneOptions) (spaceReclaimed uint64, output string, err error) {
	pruneFilters := options.filter.Value()
	pruneFilters = command.PruneFilters(dockerCli, pruneFilters)

	warning := normalWarning
	if options.all {
		warning = allCacheWarning
	}
	if !options.force && !command.PromptForConfirmation(dockerCli.In(), dockerCli.Out(), warning) {
		return 0, "", nil
	}

	report, err := dockerCli.Client().BuildCachePrune(context.Background(), types.BuildCachePruneOptions{
		All:         options.all,
		KeepStorage: options.keepStorage.Value(),
		Filters:     pruneFilters,
	})
	if err != nil {
		return 0, "", err
	}

	if len(report.CachesDeleted) > 0 {
		var sb strings.Builder
		sb.WriteString("Deleted build cache objects:\n")
		for _, id := range report.CachesDeleted {
			sb.WriteString(id)
			sb.WriteByte('\n')
		}
		output = sb.String()
	}

	return report.SpaceReclaimed, output, nil
}

// CachePrune executes a prune command for build cache
func CachePrune(dockerCli command.Cli, all bool, filter opts.FilterOpt) (uint64, string, error) {
	return runPrune(dockerCli, pruneOptions{force: true, all: all, filter: filter})
}
