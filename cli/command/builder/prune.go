package builder

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/prompt"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
)

type pruneOptions struct {
	force       bool
	all         bool
	filter      opts.FilterOpt
	keepStorage opts.MemBytes
}

// NewPruneCommand returns a new cobra prune command for images
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewPruneCommand(dockerCli command.Cli) *cobra.Command {
	return newPruneCommand(dockerCli)
}

// newPruneCommand returns a new cobra prune command for images
func newPruneCommand(dockerCLI command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove build cache",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceReclaimed, output, err := runPrune(cmd.Context(), dockerCLI, options)
			if err != nil {
				return err
			}
			if output != "" {
				_, _ = fmt.Fprintln(dockerCLI.Out(), output)
			}
			_, _ = fmt.Fprintln(dockerCLI.Out(), "Total reclaimed space:", units.HumanSize(float64(spaceReclaimed)))
			return nil
		},
		Annotations:       map[string]string{"version": "1.39"},
		ValidArgsFunction: cobra.NoFileCompletions,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.force, "force", "f", false, "Do not prompt for confirmation")
	flags.BoolVarP(&options.all, "all", "a", false, "Remove all unused build cache, not just dangling ones")
	flags.Var(&options.filter, "filter", `Provide filter values (e.g. "until=24h")`)
	flags.Var(&options.keepStorage, "keep-storage", "Amount of disk space to keep for cache")

	return cmd
}

const (
	normalWarning   = `WARNING! This will remove all dangling build cache. Are you sure you want to continue?`
	allCacheWarning = `WARNING! This will remove all build cache. Are you sure you want to continue?`
)

func runPrune(ctx context.Context, dockerCli command.Cli, options pruneOptions) (spaceReclaimed uint64, output string, err error) {
	pruneFilters := options.filter.Value()
	pruneFilters = command.PruneFilters(dockerCli, pruneFilters)

	warning := normalWarning
	if options.all {
		warning = allCacheWarning
	}
	if !options.force {
		r, err := prompt.Confirm(ctx, dockerCli.In(), dockerCli.Out(), warning)
		if err != nil {
			return 0, "", err
		}
		if !r {
			return 0, "", cancelledErr{errors.New("builder prune has been cancelled")}
		}
	}

	report, err := dockerCli.Client().BuildCachePrune(ctx, build.CachePruneOptions{
		All:         options.all,
		KeepStorage: options.keepStorage.Value(), // FIXME(thaJeztah): rewrite to use new options; see https://github.com/moby/moby/pull/48720
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

type cancelledErr struct{ error }

func (cancelledErr) Cancelled() {}

// CachePrune executes a prune command for build cache
func CachePrune(ctx context.Context, dockerCli command.Cli, all bool, filter opts.FilterOpt) (uint64, string, error) {
	return runPrune(ctx, dockerCli, pruneOptions{force: true, all: all, filter: filter})
}
