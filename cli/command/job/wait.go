package job

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/spf13/cobra"
)

// Exit statuses of a run stopped by the daemon, the same as `docker compose
// run` gives for the same run: 124 follows timeout(1), 130 the SIGINT
// convention. A script can then tell them apart from a job that failed with
// `exit 1`, without parsing stderr.
const (
	exitStatusTimedOut  = 124
	exitStatusCancelled = 130
)

// waitOptions holds the parsed flags/args of `docker job wait`.
type waitOptions struct {
	job       string
	run       string
	condition string
}

// newWaitCommand wires the `docker job wait [OPTIONS] JOB` cobra command.
// client is injected explicitly, following the same pattern as
// newCancelCommand (see cancel.go). An empty --run or --condition defers to
// the daemon's defaults ("latest"/"terminal").
func newWaitCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	var opts waitOptions

	cmd := &cobra.Command{
		Use:   "wait [OPTIONS] JOB",
		Short: "Block until a job's run reaches a terminal state",
		Long: `Block until a run of a job reaches a terminal state, then exit with the run's exit code.

The job must exist: names of removed jobs do not resolve. Without --run, the
latest run is awaited; for a schedule job that has not run yet, that means
blocking until its next scheduled fire.

A run stopped by a timeout makes the command fail with status 124, and one
stopped by "docker job cancel" with status 130, whatever the exit code of its
container. A run whose container never started fails with status 1.

With --condition running, the ID of the run is printed once it is running. A
run that already finished is reported as usual, by its exit code, so scripts
capturing the output must check which one they got.`,
		Args: cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.job = args[0]
			return runWait(cmd.Context(), dockerCli, client, &opts)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.run, "run", "", `Run to wait for: a run ID, or "latest" (default "latest")`)
	flags.StringVar(&opts.condition, "condition", "", `Condition to wait for: "terminal" or "running", i.e. the run left pending (default "terminal")`)
	return cmd
}

// runWait blocks until opts.job's run reaches opts.condition via client.Wait.
// A finished run's exit code is printed on dockerCli.Out() and mirrored into
// the process exit status via cli.StatusError, like cli/command/container/
// attach.go's getExitStatus (a zero exit code returns nil). A running run (the
// "running" condition) has no exit code yet: its ID is printed instead, like
// `docker job run`. A run stopped by a timeout or a cancellation fails with
// status 124 or 130, and one whose container never started with status 1,
// with the reason on dockerCli.Err(): an exit code, when any, is the
// container's reaction to being stopped, not the job's outcome.
func runWait(ctx context.Context, dockerCli command.Cli, client jobsClient, opts *waitOptions) error {
	reply, err := client.Wait(ctx, &jobsv0.WaitRequest{JobRef: opts.job, RunRef: opts.run, Condition: opts.condition})
	if err != nil {
		return wrapJobsError(err)
	}

	run := reply.Run
	switch {
	case run.State == jobsv0.RunStateRunning:
		_, _ = fmt.Fprintln(dockerCli.Out(), run.ID)
		return nil
	case run.State == jobsv0.RunStateTimedOut:
		_, _ = fmt.Fprintf(dockerCli.Err(), "run %s\n", run.State)
		return cli.StatusError{StatusCode: exitStatusTimedOut}
	case run.State == jobsv0.RunStateCancelled:
		_, _ = fmt.Fprintf(dockerCli.Err(), "run %s\n", run.State)
		return cli.StatusError{StatusCode: exitStatusCancelled}
	case run.ExitCode == nil:
		reason := run.Error
		if reason == "" {
			reason = run.State
		}
		_, _ = fmt.Fprintf(dockerCli.Err(), "run never started a container: %s\n", reason)
		return cli.StatusError{StatusCode: 1}
	}

	_, _ = fmt.Fprintln(dockerCli.Out(), run.ExitCode.Value)
	if run.ExitCode.Value == 0 {
		return nil
	}
	return cli.StatusError{StatusCode: int(run.ExitCode.Value)}
}
