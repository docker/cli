package job

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter/tabwriter"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

// jobRow is one formatted row of `docker job ls`, matching the columns
// NAME  TRIGGER  STATE  PAUSED  LAST RUN  EXIT  NEXT FIRE.
type jobRow struct {
	Name     string
	Trigger  string
	State    string
	Paused   string
	LastRun  string
	Exit     string
	NextFire string
}

// newJobRow formats j into a jobRow. now is the reference time used to
// humanize durations (LAST RUN age, NEXT FIRE delay) with
// github.com/docker/go-units HumanDuration; production code passes
// time.Now(), tests inject a fixed value to stay deterministic.
func newJobRow(j jobsv0.Job, now time.Time) jobRow {
	row := jobRow{
		Name:     j.Name,
		Trigger:  jobTrigger(j),
		State:    j.State,
		Paused:   strconv.FormatBool(j.Paused),
		LastRun:  "-",
		Exit:     "-",
		NextFire: "-",
	}

	if j.LatestRun != nil {
		row.LastRun = fmt.Sprintf("%s (%s ago)", j.LatestRun.State, units.HumanDuration(now.Sub(runAge(j.LatestRun))))
		if j.LatestRun.ExitCode != nil {
			row.Exit = strconv.FormatInt(j.LatestRun.ExitCode.Value, 10)
		}
	}

	if !j.Paused && j.NextFireAtNano != 0 {
		row.NextFire = "in " + units.HumanDuration(time.Unix(0, j.NextFireAtNano).Sub(now))
	}

	return row
}

// jobTrigger returns the manual shorthand unless the job declares a schedule
// trigger, in which case it returns the raw cron expression.
func jobTrigger(j jobsv0.Job) string {
	if isScheduled(&j) {
		return j.Spec.Trigger.Schedule.Cron
	}
	return jobsv0.TriggerKindManual
}

// isScheduled reports whether j declares a schedule trigger, as opposed to
// the manual shorthand.
func isScheduled(j *jobsv0.Job) bool {
	return j.Spec != nil && j.Spec.Trigger != nil && j.Spec.Trigger.Schedule != nil
}

// runAge returns the timestamp to compute a run's age from, in priority
// order: FinishedAtNano, StartedAtNano, CreatedAtNano.
func runAge(run *jobsv0.Run) time.Time {
	nano := run.FinishedAtNano
	if nano == 0 {
		nano = run.StartedAtNano
	}
	if nano == 0 {
		nano = run.CreatedAtNano
	}
	return time.Unix(0, nano)
}

// newListCommand wires the `docker job ls` cobra command. client is injected
// explicitly (rather than dialed from dockerCli internally) so that it can be
// swapped for a fake in tests without touching gRPC; the real client is the
// lazy one built by the `docker job` parent command (see cmd.go) and threaded
// down to each subcommand constructor.
func newListCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	return &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List jobs",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), dockerCli, client)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}
}

// runList calls client.List, sorts the result by Name, and renders it as a
// tabwriter table to dockerCli.Out().
func runList(ctx context.Context, dockerCli command.Cli, client jobsClient) error {
	reply, err := client.List(ctx, &jobsv0.ListRequest{})
	if err != nil {
		return wrapJobsError(err)
	}

	jobs := reply.Jobs
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].Name < jobs[j].Name
	})

	now := time.Now()
	w := tabwriter.NewWriter(dockerCli.Out(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tTRIGGER\tSTATE\tPAUSED\tLAST RUN\tEXIT\tNEXT FIRE")
	for _, j := range jobs {
		row := newJobRow(j, now)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Name, row.Trigger, row.State, row.Paused, row.LastRun, row.Exit, row.NextFire)
	}
	return w.Flush()
}
