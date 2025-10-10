package builder

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/system/pruner"
	"github.com/docker/cli/internal/prompt"
	"github.com/docker/cli/opts"
	"github.com/docker/go-units"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

func init() {
	// Register the prune command to run as part of "docker system prune"
	if err := pruner.Register(pruner.TypeBuildCache, pruneFn); err != nil {
		panic(err)
	}
}

type pruneOptions struct {
	force         bool
	all           bool
	filter        opts.FilterOpt
	reservedSpace opts.MemBytes
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
		Annotations:           map[string]string{"version": "1.39"},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.force, "force", "f", false, "Do not prompt for confirmation")
	flags.BoolVarP(&options.all, "all", "a", false, "Remove all unused build cache, not just dangling ones")
	flags.Var(&options.filter, "filter", `Provide filter values (e.g. "until=24h")`)
	flags.Var(&options.reservedSpace, "keep-storage", "Amount of disk space to keep for cache")

	return cmd
}

const (
	normalWarning   = `WARNING! This will remove all dangling build cache. Are you sure you want to continue?`
	allCacheWarning = `WARNING! This will remove all build cache. Are you sure you want to continue?`
)

func runPrune(ctx context.Context, dockerCli command.Cli, options pruneOptions) (spaceReclaimed uint64, output string, err error) {
	pruneFilters := command.PruneFilters(dockerCli, options.filter.Value())

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

	resp, err := dockerCli.Client().BuildCachePrune(ctx, client.BuildCachePruneOptions{
		All:           options.all,
		ReservedSpace: options.reservedSpace.Value(),
		Filters:       pruneFilters,
	})
	if err != nil {
		return 0, "", err
	}
	report := resp.Report
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

// pruneFn prunes the build cache for use in "docker system prune" and
// returns the amount of space reclaimed and a detailed output string.
func pruneFn(ctx context.Context, dockerCLI command.Cli, options pruner.PruneOptions) (uint64, string, error) {
	if !options.Confirmed {
		// Dry-run: perform validation and produce confirmation before pruning.
		var confirmMsg string
		if options.All {
			confirmMsg = "all build cache"
		} else {
			confirmMsg = "unused build cache"
		}
		return 0, confirmMsg, cancelledErr{errors.New("builder prune has been cancelled")}
	}
	return runPrune(ctx, dockerCLI, pruneOptions{
		force:  true,
		all:    options.All,
		filter: options.Filter,
	})
}
