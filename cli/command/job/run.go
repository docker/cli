package job

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/spf13/cobra"
)

// runOptions holds the parsed flags/args of `docker job run`.
type runOptions struct {
	job        string
	reschedule bool
}

// newRunCommand wires the `docker job run [--reschedule] JOB` cobra command.
// client is injected explicitly, following the same pattern as
// newListCommand/newLogsCommand (see list.go/logs.go).
func newRunCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	var opts runOptions

	cmd := &cobra.Command{
		Use:   "run [OPTIONS] JOB",
		Short: "Run a job",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.job = args[0]
			return runRun(cmd.Context(), dockerCli, client, &opts)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.reschedule, "reschedule", false, "Reschedule the job's next occurrence from this run")
	return cmd
}

// runRun triggers a run of opts.job via client.Run, printing the created
// run's ID on success -- the same convention as `docker container commit`
// (see cli/command/container/commit.go).
func runRun(ctx context.Context, dockerCli command.Cli, client jobsClient, opts *runOptions) error {
	reply, err := client.Run(ctx, &jobsv0.RunRequest{JobRef: opts.job, Reschedule: opts.reschedule})
	if err != nil {
		return wrapJobsError(err)
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), reply.Run.ID)
	return nil
}
