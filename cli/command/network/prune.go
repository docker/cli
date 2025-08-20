package network

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/prompt"
	"github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

type pruneOptions struct {
	force  bool
	filter opts.FilterOpt
}

// NewPruneCommand returns a new cobra prune command for networks
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewPruneCommand(dockerCLI command.Cli) *cobra.Command {
	return newPruneCommand(dockerCLI)
}

// newPruneCommand returns a new cobra prune command for networks
func newPruneCommand(dockerCLI command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove all unused networks",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := runPrune(cmd.Context(), dockerCLI, options)
			if err != nil {
				return err
			}
			if output != "" {
				_, _ = fmt.Fprintln(dockerCLI.Out(), output)
			}
			return nil
		},
		Annotations: map[string]string{"version": "1.25"},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.force, "force", "f", false, "Do not prompt for confirmation")
	flags.Var(&options.filter, "filter", `Provide filter values (e.g. "until=<timestamp>")`)

	return cmd
}

const warning = `WARNING! This will remove all custom networks not used by at least one container.
Are you sure you want to continue?`

func runPrune(ctx context.Context, dockerCli command.Cli, options pruneOptions) (output string, err error) {
	pruneFilters := command.PruneFilters(dockerCli, options.filter.Value())

	if !options.force {
		r, err := prompt.Confirm(ctx, dockerCli.In(), dockerCli.Out(), warning)
		if err != nil {
			return "", err
		}
		if !r {
			return "", cancelledErr{errors.New("network prune has been cancelled")}
		}
	}

	report, err := dockerCli.Client().NetworksPrune(ctx, pruneFilters)
	if err != nil {
		return "", err
	}

	if len(report.NetworksDeleted) > 0 {
		output = "Deleted Networks:\n"
		for _, id := range report.NetworksDeleted {
			output += id + "\n"
		}
	}

	return output, nil
}

type cancelledErr struct{ error }

func (cancelledErr) Cancelled() {}

// RunPrune calls the Network Prune API
// This returns the amount of space reclaimed and a detailed output string
func RunPrune(ctx context.Context, dockerCli command.Cli, _ bool, filter opts.FilterOpt) (uint64, string, error) {
	output, err := runPrune(ctx, dockerCli, pruneOptions{force: true, filter: filter})
	return 0, output, err
}
