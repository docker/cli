package job

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/formatter/tabwriter"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

// runRow is one formatted row of `docker job runs`, matching the columns
// ITERATION  STATE  EXIT  TRIGGER  STARTED  FINISHED  CONTAINER.
type runRow struct {
	Iteration string
	State     string
	Exit      string
	Trigger   string
	Started   string
	Finished  string
	Container string
}

// newRunRow formats r into a runRow. now is the reference time used to
// humanize STARTED/FINISHED with github.com/docker/go-units HumanDuration;
// production code passes time.Now(), tests inject a fixed value to stay
// deterministic.
func newRunRow(r jobsv0.Run, now time.Time) runRow {
	row := runRow{
		Iteration: strconv.FormatUint(r.Iteration, 10),
		State:     r.State,
		Exit:      "-",
		Trigger:   "-",
		Started:   "-",
		Finished:  "-",
		Container: "-",
	}

	if r.ExitCode != nil {
		row.Exit = strconv.FormatInt(r.ExitCode.Value, 10)
	}
	if r.Trigger != nil {
		row.Trigger = r.Trigger.Kind
	}
	if r.StartedAtNano != 0 {
		row.Started = units.HumanDuration(now.Sub(time.Unix(0, r.StartedAtNano))) + " ago"
	}
	if r.FinishedAtNano != 0 {
		row.Finished = units.HumanDuration(now.Sub(time.Unix(0, r.FinishedAtNano))) + " ago"
	}
	if r.ContainerID != "" {
		row.Container = formatter.TruncateID(r.ContainerID)
	}

	return row
}

// runsOptions holds the parsed flags/args of `docker job runs`.
type runsOptions struct {
	job    string
	limit  int32
	before string
}

// newRunsCommand wires the `docker job runs [OPTIONS] JOB` cobra command.
// client is injected explicitly, following the same pattern as
// newListCommand (see list.go).
func newRunsCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	var opts runsOptions

	cmd := &cobra.Command{
		Use:   "runs [OPTIONS] JOB",
		Short: "List a job's runs",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.job = args[0]
			return runRuns(cmd.Context(), dockerCli, client, &opts)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.Int32Var(&opts.limit, "limit", 0, "Maximum number of runs to list (0 for the daemon default)")
	flags.StringVar(&opts.before, "before", "", "List runs older than this cursor (from a previous page's NextCursor)")
	return cmd
}

// runRuns calls client.ListRuns and renders the returned page as a
// tabwriter table to dockerCli.Out(), in the order the daemon returned
// them (most-recent-first, per ListRuns's godoc) without re-sorting. A stale
// requested cursor is warned about on dockerCli.Err(); a non-empty
// NextCursor prints a paging hint on dockerCli.Out() after the table.
func runRuns(ctx context.Context, dockerCli command.Cli, client jobsClient, opts *runsOptions) error {
	reply, err := client.ListRuns(ctx, &jobsv0.ListRunsRequest{
		JobRef: opts.job,
		Limit:  opts.limit,
		Before: opts.before,
	})
	if err != nil {
		return wrapJobsError(err)
	}

	if reply.CursorStale {
		_, _ = fmt.Fprintln(dockerCli.Err(), "Warning: the requested cursor is stale; showing the most recent runs instead")
	}

	now := time.Now()
	w := tabwriter.NewWriter(dockerCli.Out(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ITERATION\tSTATE\tEXIT\tTRIGGER\tSTARTED\tFINISHED\tCONTAINER")
	for _, r := range reply.Runs {
		row := newRunRow(r, now)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Iteration, row.State, row.Exit, row.Trigger, row.Started, row.Finished, row.Container)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	if reply.NextCursor != "" {
		_, _ = fmt.Fprintf(dockerCli.Out(), "\nFor the next page, run: docker job runs --before %s %s\n", reply.NextCursor, opts.job)
	}
	return nil
}
