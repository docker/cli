package job

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/cli/internal/test"
	units "github.com/docker/go-units"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// runsHeaderColumns is the exact header line expected on `docker job runs`: ITERATION STATE EXIT TRIGGER STARTED
// FINISHED CONTAINER.
var runsHeaderColumns = []string{"ITERATION", "STATE", "EXIT", "TRIGGER", "STARTED", "FINISHED", "CONTAINER"}

func TestNewRunRow(t *testing.T) {
	now := time.Date(2024, 3, 10, 15, 0, 0, 0, time.UTC)

	testCases := []struct {
		doc      string
		run      jobsv0.Run
		expected runRow
	}{
		{
			doc: "pending run: never started, never finished, no container, no exit code",
			run: jobsv0.Run{
				Iteration: 1,
				State:     jobsv0.RunStatePending,
			},
			expected: runRow{
				Iteration: "1",
				State:     "pending",
				Exit:      "-",
				Trigger:   "-",
				Started:   "-",
				Finished:  "-",
				Container: "-",
			},
		},
		{
			doc: "running run: started, not finished, ExitCode nil, manual trigger",
			run: jobsv0.Run{
				Iteration:     2,
				State:         jobsv0.RunStateRunning,
				ContainerID:   "abcdef0123456789abcdef",
				StartedAtNano: now.Add(-30 * time.Second).UnixNano(),
				Trigger:       &jobsv0.TriggerEvidence{Kind: jobsv0.TriggerKindManual},
			},
			expected: runRow{
				Iteration: "2",
				State:     "running",
				Exit:      "-",
				Trigger:   "manual",
				Started:   "30 seconds ago",
				Finished:  "-",
				Container: "abcdef012345",
			},
		},
		{
			doc: "succeeded run with exit code zero: EXIT must be 0, NOT - (no-exit-code vs exit-code-zero trap)",
			run: jobsv0.Run{
				Iteration:      3,
				State:          jobsv0.RunStateSucceeded,
				ContainerID:    "1234567890ab1234567890ab",
				StartedAtNano:  now.Add(-5 * time.Minute).UnixNano(),
				FinishedAtNano: now.Add(-2 * time.Minute).UnixNano(),
				ExitCode:       &jobsv0.ExitCode{Value: 0},
				Trigger:        &jobsv0.TriggerEvidence{Kind: jobsv0.TriggerKindSchedule},
			},
			expected: runRow{
				Iteration: "3",
				State:     "succeeded",
				Exit:      "0",
				Trigger:   "schedule",
				Started:   "5 minutes ago",
				Finished:  "2 minutes ago",
				Container: "1234567890ab",
			},
		},
		{
			doc: "failed run with a nonzero exit code",
			run: jobsv0.Run{
				Iteration:      4,
				State:          jobsv0.RunStateFailed,
				ContainerID:    "deadbeefdeadbeefdeadbeef",
				StartedAtNano:  now.Add(-1 * time.Hour).UnixNano(),
				FinishedAtNano: now.Add(-59 * time.Minute).UnixNano(),
				ExitCode:       &jobsv0.ExitCode{Value: 137},
				Trigger:        &jobsv0.TriggerEvidence{Kind: jobsv0.TriggerKindManual},
			},
			expected: runRow{
				Iteration: "4",
				State:     "failed",
				Exit:      "137",
				Trigger:   "manual",
				Started:   "About an hour ago",
				Finished:  "59 minutes ago",
				Container: "deadbeefdead",
			},
		},
		{
			doc: "run whose container was never created: failed straight from pending, no exit code, no container",
			run: jobsv0.Run{
				Iteration:      5,
				State:          jobsv0.RunStateFailed,
				FinishedAtNano: now.Add(-10 * time.Second).UnixNano(),
				Error:          "container create failed",
			},
			expected: runRow{
				Iteration: "5",
				State:     "failed",
				Exit:      "-",
				Trigger:   "-",
				Started:   "-",
				Finished:  "10 seconds ago",
				Container: "-",
			},
		},
		{
			doc: "nil Trigger is handled defensively even though it is not expected in practice",
			run: jobsv0.Run{
				Iteration: 6,
				State:     jobsv0.RunStateCancelled,
				Trigger:   nil,
			},
			expected: runRow{
				Iteration: "6",
				State:     "cancelled",
				Exit:      "-",
				Trigger:   "-",
				Started:   "-",
				Finished:  "-",
				Container: "-",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			row := newRunRow(tc.run, now)
			assert.Check(t, is.DeepEqual(row, tc.expected))
		})
	}
}

// TestNewRunRowUsesHumanDuration guards against hardcoding a human-readable
// string that only happens to match go-units' current implementation: it
// recomputes the expected string through units.HumanDuration itself, so the
// test tracks the library rather than a magic string.
func TestNewRunRowUsesHumanDuration(t *testing.T) {
	now := time.Date(2024, 3, 10, 15, 0, 0, 0, time.UTC)
	startedAgo := 42 * time.Second
	finishedAgo := 7 * time.Second

	r := jobsv0.Run{
		Iteration:      1,
		State:          jobsv0.RunStateSucceeded,
		StartedAtNano:  now.Add(-startedAgo).UnixNano(),
		FinishedAtNano: now.Add(-finishedAgo).UnixNano(),
		ExitCode:       &jobsv0.ExitCode{Value: 0},
	}

	row := newRunRow(r, now)

	assert.Check(t, is.Equal(row.Started, units.HumanDuration(startedAgo)+" ago"))
	assert.Check(t, is.Equal(row.Finished, units.HumanDuration(finishedAgo)+" ago"))
}

// TestNewRunRowContainerTruncation confirms CONTAINER reuses the repo's
// existing ID-truncation convention (formatter.TruncateID, wrapping
// github.com/moby/moby/client/pkg/stringid.TruncateID -- the same helper used
// by `docker ps` and friends) rather than a hand-rolled slice.
func TestNewRunRowContainerTruncation(t *testing.T) {
	now := time.Date(2024, 3, 10, 15, 0, 0, 0, time.UTC)
	containerID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"

	row := newRunRow(jobsv0.Run{ContainerID: containerID}, now)

	assert.Check(t, is.Equal(row.Container, formatter.TruncateID(containerID)))
}

func TestNewRunsCommandForwardsRequest(t *testing.T) {
	var gotReq *jobsv0.ListRunsRequest
	client := &fakeClient{
		listRunsFunc: func(_ context.Context, req *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			gotReq = req
			return &jobsv0.ListRunsReply{}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"--limit", "5", "--before", "cursor-123", "backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.ListRuns was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.Limit, int32(5)))
	assert.Check(t, is.Equal(gotReq.Before, "cursor-123"))
}

func TestNewRunsCommandDefaultsLimitZeroAndBeforeEmpty(t *testing.T) {
	var gotReq *jobsv0.ListRunsRequest
	client := &fakeClient{
		listRunsFunc: func(_ context.Context, req *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			gotReq = req
			return &jobsv0.ListRunsReply{}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.ListRuns was not called")
	assert.Check(t, is.Equal(gotReq.Limit, int32(0)))
	assert.Check(t, is.Equal(gotReq.Before, ""))
}

func TestNewRunsCommandRequiresExactlyOneArg(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs(nil)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "requires 1 argument")
}

func TestNewRunsCommandHeader(t *testing.T) {
	cli := test.NewFakeCli(nil)
	client := &fakeClient{}

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	out := cli.OutBuffer().String()
	for _, col := range runsHeaderColumns {
		assert.Check(t, strings.Contains(out, col), "expected header column %q in output:\n%s", col, out)
	}
}

func TestNewRunsCommandRendersRunsInReceivedOrder(t *testing.T) {
	client := &fakeClient{
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{
				{Iteration: 9, State: jobsv0.RunStateSucceeded, ExitCode: &jobsv0.ExitCode{Value: 0}},
				{Iteration: 3, State: jobsv0.RunStateFailed, ExitCode: &jobsv0.ExitCode{Value: 1}},
				{Iteration: 7, State: jobsv0.RunStateSucceeded, ExitCode: &jobsv0.ExitCode{Value: 0}},
			}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	out := cli.OutBuffer().String()
	i9 := strings.Index(out, "9")
	i3 := strings.Index(out, "3")
	i7 := strings.Index(out, "7")
	assert.Check(t, i9 >= 0 && i3 >= 0 && i7 >= 0, "all three iterations must appear in output:\n%s", out)
	// The daemon already returns most-recent-first; the CLI must NOT re-sort.
	assert.Check(t, i9 < i3, "expected iteration 9 before 3 (no CLI re-sorting):\n%s", out)
	assert.Check(t, i3 < i7, "expected iteration 3 before 7 (no CLI re-sorting):\n%s", out)
}

func TestNewRunsCommandExitCodeZeroIsNotDash(t *testing.T) {
	client := &fakeClient{
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{
				{Iteration: 1, State: jobsv0.RunStateSucceeded, ExitCode: &jobsv0.ExitCode{Value: 0}},
			}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	out := cli.OutBuffer().String()
	assert.Check(t, hasTableCell(out, "0"), "expected a table cell with the exit code 0, not '-':\n%s", out)
}

func TestNewRunsCommandCursorStaleWarnsOnStderrBeforeTable(t *testing.T) {
	client := &fakeClient{
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			return &jobsv0.ListRunsReply{
				Runs:        []jobsv0.Run{{Iteration: 1, State: jobsv0.RunStateSucceeded}},
				CursorStale: true,
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"--before", "evicted-cursor", "backup"})
	assert.NilError(t, cmd.Execute())

	errOut := cli.ErrBuffer().String()
	assert.Check(t, errOut != "", "expected a stale-cursor warning on stderr, got none")
	assert.Check(t, strings.Contains(strings.ToLower(errOut), "stale"), "expected the warning to mention the stale cursor:\n%s", errOut)

	// The command must still succeed and render the table on stdout.
	out := cli.OutBuffer().String()
	assert.Check(t, is.Contains(out, "ITERATION"))
}

func TestNewRunsCommandNoWarningWhenCursorNotStale(t *testing.T) {
	client := &fakeClient{
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			return &jobsv0.ListRunsReply{
				Runs:        []jobsv0.Run{{Iteration: 1, State: jobsv0.RunStateSucceeded}},
				CursorStale: false,
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(cli.ErrBuffer().String(), ""))
}

func TestNewRunsCommandNextCursorPrintsPagingHint(t *testing.T) {
	client := &fakeClient{
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			return &jobsv0.ListRunsReply{
				Runs:       []jobsv0.Run{{Iteration: 1, State: jobsv0.RunStateSucceeded}},
				NextCursor: "next-page-cursor",
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	out := cli.OutBuffer().String()
	assert.Check(t, strings.Contains(out, "--before"), "expected a paging hint mentioning --before:\n%s", out)
	assert.Check(t, strings.Contains(out, "next-page-cursor"), "expected the paging hint to interpolate the actual cursor value:\n%s", out)
}

func TestNewRunsCommandNoNextCursorNoPagingHint(t *testing.T) {
	client := &fakeClient{
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			return &jobsv0.ListRunsReply{
				Runs:       []jobsv0.Run{{Iteration: 1, State: jobsv0.RunStateSucceeded}},
				NextCursor: "",
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	out := cli.OutBuffer().String()
	assert.Check(t, !strings.Contains(out, "--before"), "did not expect a paging hint when NextCursor is empty:\n%s", out)
}

func TestNewRunsCommandUnimplementedIsActionable(t *testing.T) {
	client := &fakeClient{
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			return nil, status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	// The raw gRPC status dump must not leak to the user.
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewRunsCommandOtherErrorsPassThrough(t *testing.T) {
	client := &fakeClient{
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			return nil, errors.New("job not found")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "job not found")
}

// TestRunRunsDirect exercises runRuns directly (bypassing cobra arg
// parsing), following the pattern set by list.go/logs.go/run.go of testing
// the RunE body as a plain function in addition to going through the cobra
// command.
func TestRunRunsDirect(t *testing.T) {
	var gotReq *jobsv0.ListRunsRequest
	client := &fakeClient{
		listRunsFunc: func(_ context.Context, req *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			gotReq = req
			return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{
				{Iteration: 1, State: jobsv0.RunStateSucceeded, ExitCode: &jobsv0.ExitCode{Value: 0}},
			}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runRuns(context.Background(), cli, client, &runsOptions{job: "backup", limit: 5, before: "cursor-1"})

	assert.NilError(t, err)
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.Limit, int32(5)))
	assert.Check(t, is.Equal(gotReq.Before, "cursor-1"))
	assert.Check(t, is.Contains(cli.OutBuffer().String(), "ITERATION"))
	assert.Check(t, is.Contains(cli.OutBuffer().String(), strconv.FormatUint(1, 10)))
}

// --limit is an int32 on the wire: a value that does not fit must be refused
// instead of silently wrapping around (4294967296 would become 0, i.e. the
// daemon default).
func TestNewRunsCommandRejectsLimitOutOfInt32Range(t *testing.T) {
	client := &fakeClient{
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			t.Fatal("client.ListRuns must not be called with an out-of-range --limit")
			return nil, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunsCommand(cli, client)
	cmd.SetArgs([]string{"--limit", "4294967296", "backup"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	assert.ErrorContains(t, cmd.Execute(), "--limit")
}
