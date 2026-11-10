package job

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/pkg/stdcopy"
	dockerclient "github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// pollInterval is how often waitForNextRun polls the job for a new run while
// following logs across cron occurrences.
const pollInterval = 2 * time.Second

// logsOptions holds the parsed flags/args of `docker job logs`.
type logsOptions struct {
	follow bool
	job    string
}

// newLogsCommand wires the `docker job logs [-f] JOB` cobra command. client
// is injected explicitly, following the same pattern as newListCommand (see
// list.go).
func newLogsCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	var opts logsOptions

	cmd := &cobra.Command{
		Use:   "logs [OPTIONS] JOB",
		Short: "Fetch the logs of a job's most recent run",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.job = args[0]
			return runLogs(cmd.Context(), dockerCli, client, &opts)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.follow, "follow", "f", false, "Follow log output")
	return cmd
}

// runLogs resolves opts.job through client.Inspect, picks the run to read
// logs from via resolveLogsRun -- calling client.ListRuns as its listRuns
// fallback when needed -- then streams its container's logs via
// streamRunLogs. When opts.follow is set and the run's container exits
// (streamRunLogs returns after a clean EOF), runLogs polls the job via
// waitForNextRun until the next cron occurrence starts and re-attaches to
// it, so `docker job logs -f` keeps following across runs instead of
// exiting. A context cancellation (e.g. Ctrl-C) while waiting for the next
// run ends the loop cleanly, with no error surfaced to the user.
func runLogs(ctx context.Context, dockerCli command.Cli, client jobsClient, opts *logsOptions) error {
	inspectJob := func() (*jobsv0.Job, error) {
		reply, err := client.Inspect(ctx, &jobsv0.InspectRequest{JobRef: opts.job})
		if err != nil {
			return nil, err
		}
		return reply.Job, nil
	}

	job, err := inspectJob()
	if err != nil {
		return wrapJobsError(err)
	}

	// A run is visible before its container is created: right after
	// `docker job run`, the latest run is pending. Wait for it to leave
	// pending, like `docker compose run` does, rather than falling back to an
	// older run, or failing for a job that never ran before. The run is
	// waited for by ID, as another run may become the latest in the meantime.
	if job.LatestRun != nil && job.LatestRun.State == jobsv0.RunStatePending {
		reply, err := client.Wait(ctx, &jobsv0.WaitRequest{
			JobRef:    opts.job,
			RunRef:    job.LatestRun.ID,
			Condition: jobsv0.WaitConditionRunning,
		})
		if err != nil {
			return wrapJobsError(err)
		}
		job.LatestRun = reply.Run
	}

	run, err := resolveLogsRun(job, func() (*jobsv0.ListRunsReply, error) {
		return client.ListRuns(ctx, &jobsv0.ListRunsRequest{JobRef: opts.job})
	})
	if err != nil {
		return wrapJobsError(err)
	}
	if run.ID != job.LatestRun.ID {
		_, _ = fmt.Fprint(dockerCli.Err(), fallbackNote(job.LatestRun, run, time.Now()))
	}

	// Only a schedule job can produce a next run on its own.
	followNext := opts.follow && isScheduled(job)
	for {
		if err := streamRunLogs(ctx, dockerCli, run.ContainerID, opts.follow); err != nil {
			return err
		}
		if !followNext {
			return nil
		}

		run, err = waitForNextRun(ctx, run.ID, inspectJob, sleepOrCancel)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return wrapJobsError(err)
		}
	}
}

// fallbackNote tells why the logs shown are those of run shown rather than of
// the latest run, with the state and the age of both. A latest run that
// succeeded and lost its container is the normal case of a job registered
// with RemoveOnSuccess: the note says so first, as the output of an older
// failed run, shown without it, reads like the job is failing.
func fallbackNote(latest, shown *jobsv0.Run, now time.Time) string {
	if latest.State == jobsv0.RunStateSucceeded && latest.ContainerGone {
		return fmt.Sprintf("Note: the latest run %s %s and its container was removed, so its output is gone.\nShowing the logs of run %s (%s):\n",
			latest.ID, runStateAge(latest, now, " "), shown.ID, runStateAge(shown, now, ", "))
	}
	return fmt.Sprintf("Note: the container of the latest run %s (%s) is not available; showing the logs of run %s (%s)\n",
		latest.ID, runStateAge(latest, now, ", "), shown.ID, runStateAge(shown, now, ", "))
}

// runStateAge describes run as its state, followed by sep and how long ago it
// ended (or, when it has not, was created), when the daemon recorded the time.
func runStateAge(run *jobsv0.Run, now time.Time, sep string) string {
	at := run.FinishedAtNano
	if at == 0 {
		at = run.CreatedAtNano
	}
	if at == 0 {
		return run.State
	}
	return run.State + sep + units.HumanDuration(now.Sub(time.Unix(0, at))) + " ago"
}

// streamRunLogs attaches to containerID's logs and demultiplexes them onto
// dockerCli.Out()/dockerCli.Err(), following the pattern of
// cli/command/container/logs.go's runLogs. It returns nil once the log
// stream ends with a clean EOF.
func streamRunLogs(ctx context.Context, dockerCli command.Cli, containerID string, follow bool) error {
	c, err := dockerCli.Client().ContainerInspect(ctx, containerID, dockerclient.ContainerInspectOptions{})
	if errdefs.IsNotFound(err) {
		// The container was removed (history eviction, job removal) after
		// the run was looked up.
		return errors.New("logs are no longer available: the run's container was removed")
	}
	if err != nil {
		return err
	}

	resp, err := dockerCli.Client().ContainerLogs(ctx, containerID, dockerclient.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
	})
	if err != nil {
		return err
	}
	defer func() { _ = resp.Close() }()

	if c.Container.Config.Tty {
		_, err = io.Copy(dockerCli.Out(), resp)
	} else {
		_, err = stdcopy.StdCopy(dockerCli.Out(), dockerCli.Err(), resp)
	}
	return err
}

// waitForNextRun polls inspect() until job.LatestRun's ID differs from
// currentRunID and its logs can be read (see hasContainer), returning that new
// run. A run is visible before its container is created (and never gets one if
// creation fails), and a fast job's container may already be gone by the time
// it is noticed: such runs are skipped rather than returned.
//
// It checks ctx.Err() before every poll so an already-cancelled (or
// cancelled-while-sleeping) context returns immediately, and tolerates
// job.LatestRun being nil (a job that hasn't run again yet) by continuing to
// poll instead of panicking. sleep is called between polls and its error
// (e.g. context cancellation) is propagated.
func waitForNextRun(ctx context.Context, currentRunID string, inspect func() (*jobsv0.Job, error), sleep func(context.Context) error) (*jobsv0.Run, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		job, err := inspect()
		if err != nil {
			return nil, err
		}

		if job.LatestRun != nil && job.LatestRun.ID != currentRunID && hasContainer(job.LatestRun) {
			return job.LatestRun, nil
		}

		if err := sleep(ctx); err != nil {
			return nil, err
		}
	}
}

// sleepOrCancel is waitForNextRun's production sleep: it waits pollInterval
// between polls, returning early with ctx.Err() if the context is cancelled
// first.
func sleepOrCancel(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(pollInterval):
		return nil
	}
}

// resolveLogsRun picks the run whose container `docker job logs` should read
// from: job.LatestRun when its logs can be read, otherwise the most recent run
// of the job's history (via listRuns, called lazily) that still has a
// container. job is the already-inspected job; this function does no I/O of
// its own.
func resolveLogsRun(job *jobsv0.Job, listRuns func() (*jobsv0.ListRunsReply, error)) (*jobsv0.Run, error) {
	if job.LatestRun == nil {
		return nil, fmt.Errorf("job %q has never run", job.Name)
	}
	if hasContainer(job.LatestRun) {
		return job.LatestRun, nil
	}

	reply, err := listRuns()
	if err != nil {
		return nil, err
	}
	for i := range reply.Runs {
		if hasContainer(&reply.Runs[i]) {
			return &reply.Runs[i], nil
		}
	}
	return nil, fmt.Errorf("no logs available for job %q", job.Name)
}

// hasContainer reports whether run's logs can still be read: its container
// has been created and not removed.
func hasContainer(run *jobsv0.Run) bool {
	return run.ContainerID != "" && !run.ContainerGone
}
