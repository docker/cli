package job

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/cli/internal/prompt"
	"github.com/spf13/cobra"
)

// pruneOptions holds the parsed flags of `docker job prune`.
type pruneOptions struct {
	labels []string
	force  bool
}

// newPruneCommand wires the `docker job prune [--label ...]` cobra command.
// client is injected explicitly, following the same pattern as
// newRunCommand (see run.go).
func newPruneCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	var opts pruneOptions

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove idle manual jobs",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrune(cmd.Context(), dockerCli, client, &opts)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Do not prompt for confirmation")
	flags.StringArrayVar(&opts.labels, "label", nil, "Only remove jobs matching a label (\"key\" or \"key=value\")")
	return cmd
}

const pruneWarning = `WARNING! This will remove all idle manual jobs.
Are you sure you want to continue?`

// runPrune removes idle manual jobs matching opts.labels via client.Prune,
// printing each removed job's ID on its own line on dockerCli.Out() --
// mirroring `docker container prune`/`docker volume prune`, including the
// confirmation prompt that opts.force skips. Nothing removed prints nothing
// and returns a nil error, keeping the command scriptable.
func runPrune(ctx context.Context, dockerCli command.Cli, client jobsClient, opts *pruneOptions) error {
	if !opts.force {
		r, err := prompt.Confirm(ctx, dockerCli.In(), dockerCli.Out(), pruneWarning)
		if err != nil {
			return err
		}
		if !r {
			return cancelledErr{errors.New("job prune has been cancelled")}
		}
	}

	reply, err := client.Prune(ctx, &jobsv0.PruneRequest{Labels: opts.labels})
	if err != nil {
		return wrapJobsError(err)
	}

	for _, id := range reply.RemovedJobIDs {
		_, _ = fmt.Fprintln(dockerCli.Out(), id)
	}
	return nil
}

type cancelledErr struct{ error }

func (cancelledErr) Cancelled() {}
