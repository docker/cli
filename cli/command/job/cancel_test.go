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

// Output convention choice: on a successful cancellation (reply.RunID set),
// runCancel prints the cancelled run's ID on dockerCli.Out(), mirroring
// runRun's "print the created/affected resource's ID" convention (see
// run.go). When no run was in flight (reply.RunID == ""), printing an empty
// line would be confusing in a demo -- the user asked for a cancel and
// nothing happened, silently or with a blank line, which looks like a bug.
// Instead runCancel prints a clear, non-error message to dockerCli.Out()
// (`no run in flight for job "<jobRef>"`) and returns a nil error: asking to
// cancel an idle job is a valid, idempotent request, not a failure -- the
// same spirit as jobsv0.Jobs.Pause/Resume being documented idempotent.

func TestNewCancelCommandForwardsJobRef(t *testing.T) {
	var gotReq *jobsv0.CancelRequest
	client := &fakeClient{
		cancelFunc: func(_ context.Context, req *jobsv0.CancelRequest) (*jobsv0.CancelReply, error) {
			gotReq = req
			return &jobsv0.CancelReply{RunID: "run-1"}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newCancelCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Cancel was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
}

func TestNewCancelCommandPrintsCancelledRunID(t *testing.T) {
	client := &fakeClient{
		cancelFunc: func(context.Context, *jobsv0.CancelRequest) (*jobsv0.CancelReply, error) {
			return &jobsv0.CancelReply{RunID: "run-42"}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newCancelCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(cli.OutBuffer().String(), "run-42\n"))
}

func TestNewCancelCommandNoRunInFlightPrintsMessageWithoutError(t *testing.T) {
	client := &fakeClient{
		cancelFunc: func(context.Context, *jobsv0.CancelRequest) (*jobsv0.CancelReply, error) {
			return &jobsv0.CancelReply{RunID: ""}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newCancelCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(cli.OutBuffer().String(), "no run in flight for job \"backup\"\n"))
}

func TestNewCancelCommandRequiresExactlyOneArg(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(nil)

	cmd := newCancelCommand(cli, client)
	cmd.SetArgs(nil)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "requires 1 argument")
}

func TestNewCancelCommandUnimplementedIsActionable(t *testing.T) {
	client := &fakeClient{
		cancelFunc: func(context.Context, *jobsv0.CancelRequest) (*jobsv0.CancelReply, error) {
			return nil, status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newCancelCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	// The raw gRPC status dump must not leak to the user.
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewCancelCommandOtherErrorsPassThroughUnwrapped(t *testing.T) {
	errText := `job "backup" not found`
	client := &fakeClient{
		cancelFunc: func(context.Context, *jobsv0.CancelRequest) (*jobsv0.CancelReply, error) {
			return nil, errors.New(errText)
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newCancelCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, errText)
}

// TestRunCancelDirect exercises runCancel directly (bypassing cobra arg
// parsing), following the pattern set by run_test.go/inspect_test.go of
// testing the RunE body as a plain function in addition to going through the
// cobra command.
func TestRunCancelDirect(t *testing.T) {
	var gotReq *jobsv0.CancelRequest
	client := &fakeClient{
		cancelFunc: func(_ context.Context, req *jobsv0.CancelRequest) (*jobsv0.CancelReply, error) {
			gotReq = req
			return &jobsv0.CancelReply{RunID: "run-7"}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runCancel(context.Background(), cli, client, "backup")

	assert.NilError(t, err)
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "run-7\n"))
}

func TestRunCancelDirectNoRunInFlight(t *testing.T) {
	client := &fakeClient{
		cancelFunc: func(context.Context, *jobsv0.CancelRequest) (*jobsv0.CancelReply, error) {
			return &jobsv0.CancelReply{RunID: ""}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runCancel(context.Background(), cli, client, "backup")

	assert.NilError(t, err)
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "no run in flight for job \"backup\"\n"))
}
