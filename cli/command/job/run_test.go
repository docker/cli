package job

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/cli/internal/test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNewRunCommandForwardsJobRefAndDefaultsRescheduleFalse(t *testing.T) {
	var gotReq *jobsv0.RunRequest
	client := &fakeClient{
		runFunc: func(_ context.Context, req *jobsv0.RunRequest) (*jobsv0.RunReply, error) {
			gotReq = req
			return &jobsv0.RunReply{Run: &jobsv0.Run{ID: "run-1"}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Run was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.Reschedule, false))
}

func TestNewRunCommandRescheduleFlagForwarded(t *testing.T) {
	var gotReq *jobsv0.RunRequest
	client := &fakeClient{
		runFunc: func(_ context.Context, req *jobsv0.RunRequest) (*jobsv0.RunReply, error) {
			gotReq = req
			return &jobsv0.RunReply{Run: &jobsv0.Run{ID: "run-1"}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunCommand(cli, client)
	cmd.SetArgs([]string{"--reschedule", "backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Run was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.Reschedule, true))
}

func TestNewRunCommandPrintsCreatedRunID(t *testing.T) {
	client := &fakeClient{
		runFunc: func(context.Context, *jobsv0.RunRequest) (*jobsv0.RunReply, error) {
			return &jobsv0.RunReply{Run: &jobsv0.Run{ID: "run-42"}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	// Convention: print only the created resource's ID, one line, nothing
	// else -- same as `docker container commit` (see
	// cli/command/container/commit.go) and `docker run`'s container ID.
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "run-42\n"))
}

func TestNewRunCommandRequiresExactlyOneArg(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(nil)

	cmd := newRunCommand(cli, client)
	cmd.SetArgs(nil)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "requires 1 argument")
}

func TestNewRunCommandUnimplementedIsActionable(t *testing.T) {
	client := &fakeClient{
		runFunc: func(context.Context, *jobsv0.RunRequest) (*jobsv0.RunReply, error) {
			return nil, status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRunCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	// The raw gRPC status dump must not leak to the user.
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewRunCommandOtherErrorsPassThroughUnwrapped(t *testing.T) {
	testCases := []struct {
		doc     string
		errText string
	}{
		{doc: "job not found", errText: `job "backup" not found`},
		{doc: "reschedule rejected for a job without a schedule trigger", errText: "reschedule is not supported for jobs without a schedule trigger"},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			client := &fakeClient{
				runFunc: func(context.Context, *jobsv0.RunRequest) (*jobsv0.RunReply, error) {
					return nil, errors.New(tc.errText)
				},
			}
			cli := test.NewFakeCli(nil)

			cmd := newRunCommand(cli, client)
			cmd.SetArgs([]string{"backup"})
			cmd.SetOut(io.Discard)
			err := cmd.Execute()

			assert.ErrorContains(t, err, tc.errText)
		})
	}
}

// TestRunRunDirect exercises runRun directly (bypassing cobra arg parsing),
// following the pattern set by list.go/logs.go of testing the RunE body as a
// plain function in addition to going through the cobra command.
func TestRunRunDirect(t *testing.T) {
	var gotReq *jobsv0.RunRequest
	client := &fakeClient{
		runFunc: func(_ context.Context, req *jobsv0.RunRequest) (*jobsv0.RunReply, error) {
			gotReq = req
			return &jobsv0.RunReply{Run: &jobsv0.Run{ID: "run-7"}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runRun(context.Background(), cli, client, &runOptions{job: "backup", reschedule: true})

	assert.NilError(t, err)
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.Reschedule, true))
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "run-7\n"))
}
