package container

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

type pruneOptions struct {
	force  bool
	filter opts.FilterOpt
}

// NewPruneCommand returns a new cobra prune command for containers
func NewPruneCommand(dockerCli command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove all stopped containers",
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
		Annotations: map[string]string{"version": "1.25"},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.force, "force", "f", false, "Do not prompt for confirmation")
	flags.Var(&options.filter, "filter", "Provide filter values (e.g. 'until=<timestamp>')")

	return cmd
}

const warning = `WARNING! This will remove all stopped containers.
Are you sure you want to continue?`

func runPrune(dockerCli command.Cli, options pruneOptions) (spaceReclaimed uint64, output string, err error) {
	pruneFilters := command.PruneFilters(dockerCli, options.filter.Value())

	if !options.force && !command.PromptForConfirmation(dockerCli.In(), dockerCli.Out(), warning) {
		return 0, "", nil
	}

	report, err := dockerCli.Client().ContainersPrune(context.Background(), pruneFilters)
	if err != nil {
		return 0, "", err
	}

	if len(report.ContainersDeleted) > 0 {
		output = "Deleted Containers:\n"
		for _, id := range report.ContainersDeleted {
			output += id + "\n"
		}
		spaceReclaimed = report.SpaceReclaimed
	}

	return spaceReclaimed, output, nil
}

// RunPrune calls the Container Prune API
// This returns the amount of space reclaimed and a detailed output string
func RunPrune(dockerCli command.Cli, all bool, filter opts.FilterOpt) (uint64, string, error) {
	return runPrune(dockerCli, pruneOptions{force: true, filter: filter})
}
