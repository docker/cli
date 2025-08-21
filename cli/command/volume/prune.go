package volume

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/system/pruner"
	"github.com/docker/cli/internal/prompt"
	"github.com/docker/cli/opts"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/versions"
	"github.com/spf13/cobra"
)

func init() {
	// Register the prune command to run as part of "docker system prune"
	if err := pruner.Register(pruner.TypeVolume, pruneFn); err != nil {
		panic(err)
	}
}

type pruneOptions struct {
	all    bool
	force  bool
	filter opts.FilterOpt
}

// newPruneCommand returns a new cobra prune command for volumes
func newPruneCommand(dockerCLI command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove unused local volumes",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceReclaimed, output, err := runPrune(cmd.Context(), dockerCLI, options)
			if err != nil {
				return err
			}
			if output != "" {
				fmt.Fprintln(dockerCLI.Out(), output)
			}
			fmt.Fprintln(dockerCLI.Out(), "Total reclaimed space:", units.HumanSize(float64(spaceReclaimed)))
			return nil
		},
		Annotations:       map[string]string{"version": "1.25"},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.all, "all", "a", false, "Remove all unused volumes, not just anonymous ones")
	flags.SetAnnotation("all", "version", []string{"1.42"})
	flags.BoolVarP(&options.force, "force", "f", false, "Do not prompt for confirmation")
	flags.Var(&options.filter, "filter", `Provide filter values (e.g. "label=<label>")`)

	return cmd
}

const (
	unusedVolumesWarning = `WARNING! This will remove anonymous local volumes not used by at least one container.
Are you sure you want to continue?`
	allVolumesWarning = `WARNING! This will remove all local volumes not used by at least one container.
Are you sure you want to continue?`
)

func runPrune(ctx context.Context, dockerCli command.Cli, options pruneOptions) (spaceReclaimed uint64, output string, err error) {
	pruneFilters := command.PruneFilters(dockerCli, options.filter.Value())

	warning := unusedVolumesWarning
	if versions.GreaterThanOrEqualTo(dockerCli.CurrentVersion(), "1.42") {
		if options.all {
			if pruneFilters.Contains("all") {
				return 0, "", invalidParamErr{errors.New("conflicting options: cannot specify both --all and --filter all=1")}
			}
			pruneFilters.Add("all", "true")
			warning = allVolumesWarning
		}
	} else {
		// API < v1.42 removes all volumes (anonymous and named) by default.
		warning = allVolumesWarning
	}
	if !options.force {
		r, err := prompt.Confirm(ctx, dockerCli.In(), dockerCli.Out(), warning)
		if err != nil {
			return 0, "", err
		}
		if !r {
			return 0, "", cancelledErr{errors.New("volume prune has been cancelled")}
		}
	}

	report, err := dockerCli.Client().VolumesPrune(ctx, pruneFilters)
	if err != nil {
		return 0, "", err
	}

	if len(report.VolumesDeleted) > 0 {
		output = "Deleted Volumes:\n"
		for _, id := range report.VolumesDeleted {
			output += id + "\n"
		}
		spaceReclaimed = report.SpaceReclaimed
	}

	return spaceReclaimed, output, nil
}

type invalidParamErr struct{ error }

func (invalidParamErr) InvalidParameter() {}

type cancelledErr struct{ error }

func (cancelledErr) Cancelled() {}

// pruneFn calls the Volume Prune API for use in "docker system prune",
// and returns the amount of space reclaimed and a detailed output string.
func pruneFn(ctx context.Context, dockerCli command.Cli, options pruner.PruneOptions) (uint64, string, error) {
	// TODO version this once "until" filter is supported for volumes
	// Ideally, this check wasn't done on the CLI because the list of
	// filters that is supported by the daemon may evolve over time.
	if options.Filter.Value().Contains("until") {
		return 0, "", errors.New(`ERROR: The "until" filter is not supported with "--volumes"`)
	}
	if !options.Confirmed {
		// Dry-run: perform validation and produce confirmation before pruning.
		confirmMsg := "all anonymous volumes not used by at least one container"
		return 0, confirmMsg, cancelledErr{errors.New("volume prune has been cancelled")}
	}
	return runPrune(ctx, dockerCli, pruneOptions{
		force:  true,
		filter: options.Filter,
	})
}
