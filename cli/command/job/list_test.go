package job

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/cli/internal/test"
	"github.com/docker/go-units"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// jobsHeaderColumns is the exact header line expected on `docker job ls`: NAME TRIGGER STATE PAUSED LAST RUN EXIT NEXT FIRE.
var jobsHeaderColumns = []string{"NAME", "TRIGGER", "STATE", "PAUSED", "LAST RUN", "EXIT", "NEXT FIRE"}

func TestNewJobRow(t *testing.T) {
	now := time.Date(2024, 3, 10, 15, 0, 0, 0, time.UTC)

	testCases := []struct {
		doc      string
		job      jobsv0.Job
		expected jobRow
	}{
		{
			doc: "manual job, never run",
			job: jobsv0.Job{
				Name:  "migrate",
				State: jobsv0.JobStateIdle,
				Spec: &jobsv0.JobSpec{
					Trigger: &jobsv0.Trigger{Manual: true},
				},
			},
			expected: jobRow{
				Name:     "migrate",
				Trigger:  "manual",
				State:    "idle",
				Paused:   "false",
				LastRun:  "-",
				Exit:     "-",
				NextFire: "-",
			},
		},
		{
			doc: "nil Spec is the manual shorthand",
			job: jobsv0.Job{
				Name:  "migrate",
				State: jobsv0.JobStateIdle,
				Spec:  nil,
			},
			expected: jobRow{
				Name:     "migrate",
				Trigger:  "manual",
				State:    "idle",
				Paused:   "false",
				LastRun:  "-",
				Exit:     "-",
				NextFire: "-",
			},
		},
		{
			doc: "nil Trigger on a Spec is also the manual shorthand",
			job: jobsv0.Job{
				Name:  "migrate",
				State: jobsv0.JobStateIdle,
				Spec:  &jobsv0.JobSpec{Trigger: nil},
			},
			expected: jobRow{
				Name:     "migrate",
				Trigger:  "manual",
				State:    "idle",
				Paused:   "false",
				LastRun:  "-",
				Exit:     "-",
				NextFire: "-",
			},
		},
		{
			doc: "cron job, armed, not paused",
			job: jobsv0.Job{
				Name:  "backup",
				State: jobsv0.JobStateIdle,
				Spec: &jobsv0.JobSpec{
					Trigger: &jobsv0.Trigger{
						Schedule: &jobsv0.ScheduleTrigger{Cron: "*/5 * * * *"},
					},
				},
				NextFireAtNano: now.Add(5 * time.Minute).UnixNano(),
			},
			expected: jobRow{
				Name:     "backup",
				Trigger:  "*/5 * * * *",
				State:    "idle",
				Paused:   "false",
				LastRun:  "-",
				Exit:     "-",
				NextFire: "in 5 minutes",
			},
		},
		{
			doc: "paused cron job with a stale NextFireAtNano must show NEXT FIRE -, not a computed value",
			job: jobsv0.Job{
				Name:   "backup",
				State:  jobsv0.JobStateIdle,
				Paused: true,
				Spec: &jobsv0.JobSpec{
					Trigger: &jobsv0.Trigger{
						Schedule: &jobsv0.ScheduleTrigger{Cron: "* * * * *"},
					},
				},
				// Residual/stale value from before the job was paused: must
				// be ignored, not rendered.
				NextFireAtNano: now.Add(3 * time.Minute).UnixNano(),
			},
			expected: jobRow{
				Name:     "backup",
				Trigger:  "* * * * *",
				State:    "idle",
				Paused:   "true",
				LastRun:  "-",
				Exit:     "-",
				NextFire: "-",
			},
		},
		{
			doc: "unpaused cron job currently running has NextFireAtNano == 0 -> NEXT FIRE -",
			job: jobsv0.Job{
				Name:  "backup",
				State: jobsv0.JobStateRunning,
				Spec: &jobsv0.JobSpec{
					Trigger: &jobsv0.Trigger{
						Schedule: &jobsv0.ScheduleTrigger{Cron: "* * * * *"},
					},
				},
				NextFireAtNano: 0,
			},
			expected: jobRow{
				Name:     "backup",
				Trigger:  "* * * * *",
				State:    "running",
				Paused:   "false",
				LastRun:  "-",
				Exit:     "-",
				NextFire: "-",
			},
		},
		{
			doc: "job never ran: LAST RUN and EXIT are both -",
			job: jobsv0.Job{
				Name:      "migrate",
				State:     jobsv0.JobStateIdle,
				Spec:      &jobsv0.JobSpec{Trigger: &jobsv0.Trigger{Manual: true}},
				LatestRun: nil,
			},
			expected: jobRow{
				Name:     "migrate",
				Trigger:  "manual",
				State:    "idle",
				Paused:   "false",
				LastRun:  "-",
				Exit:     "-",
				NextFire: "-",
			},
		},
		{
			doc: "run in flight: ExitCode nil -> EXIT -, LAST RUN uses StartedAtNano since FinishedAtNano is zero",
			job: jobsv0.Job{
				Name:  "backup",
				State: jobsv0.JobStateRunning,
				Spec:  &jobsv0.JobSpec{Trigger: &jobsv0.Trigger{Manual: true}},
				LatestRun: &jobsv0.Run{
					State:         jobsv0.RunStateRunning,
					StartedAtNano: now.Add(-30 * time.Second).UnixNano(),
					ExitCode:      nil,
				},
			},
			expected: jobRow{
				Name:     "backup",
				Trigger:  "manual",
				State:    "running",
				Paused:   "false",
				LastRun:  "running (30 seconds ago)",
				Exit:     "-",
				NextFire: "-",
			},
		},
		{
			doc: "successful run with exit code zero: EXIT must be 0, NOT - (no-exit-code vs exit-code-zero trap)",
			job: jobsv0.Job{
				Name:  "backup",
				State: jobsv0.JobStateIdle,
				Spec:  &jobsv0.JobSpec{Trigger: &jobsv0.Trigger{Manual: true}},
				LatestRun: &jobsv0.Run{
					State:          jobsv0.RunStateSucceeded,
					FinishedAtNano: now.Add(-2 * time.Minute).UnixNano(),
					ExitCode:       &jobsv0.ExitCode{Value: 0},
				},
			},
			expected: jobRow{
				Name:     "backup",
				Trigger:  "manual",
				State:    "idle",
				Paused:   "false",
				LastRun:  "succeeded (2 minutes ago)",
				Exit:     "0",
				NextFire: "-",
			},
		},
		{
			doc: "failed run with a nonzero exit code",
			job: jobsv0.Job{
				Name:  "backup",
				State: jobsv0.JobStateIdle,
				Spec:  &jobsv0.JobSpec{Trigger: &jobsv0.Trigger{Manual: true}},
				LatestRun: &jobsv0.Run{
					State:          jobsv0.RunStateFailed,
					FinishedAtNano: now.Add(-1 * time.Hour).UnixNano(),
					ExitCode:       &jobsv0.ExitCode{Value: 137},
				},
			},
			expected: jobRow{
				Name:     "backup",
				Trigger:  "manual",
				State:    "idle",
				Paused:   "false",
				LastRun:  "failed (About an hour ago)",
				Exit:     "137",
				NextFire: "-",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			row := newJobRow(tc.job, now)
			assert.Check(t, is.DeepEqual(row, tc.expected))
		})
	}
}

// TestNewJobRowNextFireUsesHumanDuration guards against hardcoding a
// human-readable string that only happens to match go-units' current
// implementation: it recomputes the expected string through
// units.HumanDuration itself, so the test tracks the library rather than a
// magic string.
func TestNewJobRowNextFireUsesHumanDuration(t *testing.T) {
	now := time.Date(2024, 3, 10, 15, 0, 0, 0, time.UTC)
	delay := 42 * time.Second
	j := jobsv0.Job{
		Name: "backup",
		Spec: &jobsv0.JobSpec{
			Trigger: &jobsv0.Trigger{Schedule: &jobsv0.ScheduleTrigger{Cron: "* * * * *"}},
		},
		NextFireAtNano: now.Add(delay).UnixNano(),
	}

	row := newJobRow(j, now)

	assert.Check(t, is.Equal(row.NextFire, "in "+units.HumanDuration(delay)))
}

func TestNewListCommandSortsByName(t *testing.T) {
	cli := test.NewFakeCli(nil)
	client := &fakeClient{
		listFunc: func(context.Context, *jobsv0.ListRequest) (*jobsv0.ListReply, error) {
			return &jobsv0.ListReply{Jobs: []jobsv0.Job{
				{Name: "zeta", Spec: &jobsv0.JobSpec{Trigger: &jobsv0.Trigger{Manual: true}}},
				{Name: "alpha", Spec: &jobsv0.JobSpec{Trigger: &jobsv0.Trigger{Manual: true}}},
				{Name: "mike", Spec: &jobsv0.JobSpec{Trigger: &jobsv0.Trigger{Manual: true}}},
			}}, nil
		},
	}

	cmd := newListCommand(cli, client)
	cmd.SetOut(io.Discard)
	assert.NilError(t, cmd.Execute())

	out := cli.OutBuffer().String()
	iAlpha := strings.Index(out, "alpha")
	iMike := strings.Index(out, "mike")
	iZeta := strings.Index(out, "zeta")
	assert.Check(t, iAlpha >= 0 && iMike >= 0 && iZeta >= 0, "all three job names must appear in output:\n%s", out)
	assert.Check(t, iAlpha < iMike, "expected alpha before mike:\n%s", out)
	assert.Check(t, iMike < iZeta, "expected mike before zeta:\n%s", out)
}

func TestNewListCommandHeader(t *testing.T) {
	cli := test.NewFakeCli(nil)
	client := &fakeClient{}

	cmd := newListCommand(cli, client)
	assert.NilError(t, cmd.Execute())

	out := cli.OutBuffer().String()
	for _, col := range jobsHeaderColumns {
		assert.Check(t, strings.Contains(out, col), "expected header column %q in output:\n%s", col, out)
	}
}

func TestNewListCommandRendersJobFields(t *testing.T) {
	cli := test.NewFakeCli(nil)
	client := &fakeClient{
		listFunc: func(context.Context, *jobsv0.ListRequest) (*jobsv0.ListReply, error) {
			return &jobsv0.ListReply{Jobs: []jobsv0.Job{
				{
					Name:   "backup",
					State:  jobsv0.JobStateIdle,
					Paused: true,
					Spec: &jobsv0.JobSpec{
						Trigger: &jobsv0.Trigger{Schedule: &jobsv0.ScheduleTrigger{Cron: "* * * * *"}},
					},
					LatestRun: &jobsv0.Run{
						State:    jobsv0.RunStateSucceeded,
						ExitCode: &jobsv0.ExitCode{Value: 0},
					},
				},
			}}, nil
		},
	}

	cmd := newListCommand(cli, client)
	assert.NilError(t, cmd.Execute())

	out := cli.OutBuffer().String()
	assert.Check(t, is.Contains(out, "backup"))
	assert.Check(t, is.Contains(out, "* * * * *"))
	assert.Check(t, is.Contains(out, "idle"))
	assert.Check(t, is.Contains(out, "true")) // Paused column
	assert.Check(t, is.Contains(out, "succeeded"))
	// The exit code column must show the literal 0, not a "-" placeholder.
	assert.Check(t, hasTableCell(out, "0"), "expected a table cell with the exit code 0, not '-':\n%s", out)
}

func TestNewListCommandUnimplementedIsActionable(t *testing.T) {
	cli := test.NewFakeCli(nil)
	client := &fakeClient{
		listFunc: func(context.Context, *jobsv0.ListRequest) (*jobsv0.ListReply, error) {
			return nil, status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}

	cmd := newListCommand(cli, client)
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	// The raw gRPC status dump must not leak to the user.
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewListCommandOtherErrorsPassThrough(t *testing.T) {
	cli := test.NewFakeCli(nil)
	client := &fakeClient{
		listFunc: func(context.Context, *jobsv0.ListRequest) (*jobsv0.ListReply, error) {
			return nil, errors.New("connection refused")
		},
	}

	cmd := newListCommand(cli, client)
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "connection refused")
}

// hasTableCell reports whether out contains a whitespace-delimited cell
// equal to want. It exists because a plain strings.Contains(out, "0") would
// also match inside unrelated substrings (job names, cron fields, etc.).
func hasTableCell(out, want string) bool {
	for _, field := range strings.Fields(out) {
		if field == want {
			return true
		}
	}
	return false
}
