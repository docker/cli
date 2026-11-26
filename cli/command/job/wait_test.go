package job

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	clipkg "github.com/docker/cli/cli"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/cli/internal/test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// Unlike `docker container wait` (cli/command/container/wait.go), which only
// ever prints the exit code on stdout and never affects the CLI process's
// own exit status, `docker job wait` is meant to be scriptable
// (`docker job wait X && do-something`): the process must exit with the
// run's own exit code. That is done by returning cli.StatusError{StatusCode:
// N} from runWait, mirroring cli/command/container/attach.go's
// getExitStatus. As in that precedent, a zero exit code returns a nil error
// (not cli.StatusError{StatusCode: 0}), since StatusCode 0 combined with a
// non-nil error does not extract correctly (see the FIXME in
// cmd/docker/docker.go).

func TestNewWaitCommandForwardsJobRef(t *testing.T) {
	var gotReq *jobsv0.WaitRequest
	client := &fakeClient{
		waitFunc: func(_ context.Context, req *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			gotReq = req
			return &jobsv0.WaitReply{
				Run: &jobsv0.Run{
					State:    jobsv0.RunStateSucceeded,
					ExitCode: &jobsv0.ExitCode{Value: 0},
				},
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newWaitCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Wait was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.RunRef, ""))
	assert.Check(t, is.Equal(gotReq.Condition, ""))
}

func TestNewWaitCommandRequiresExactlyOneArg(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(nil)

	cmd := newWaitCommand(cli, client)
	cmd.SetArgs(nil)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "requires 1 argument")
}

func TestNewWaitCommandUnimplementedIsActionable(t *testing.T) {
	client := &fakeClient{
		waitFunc: func(context.Context, *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			return nil, status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newWaitCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewWaitCommandOtherErrorsPassThroughUnwrapped(t *testing.T) {
	errText := `job "backup" not found`
	client := &fakeClient{
		waitFunc: func(context.Context, *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			return nil, errors.New(errText)
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newWaitCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, errText)
}

// TestRunWaitDirect* exercise runWait directly (bypassing cobra arg
// parsing), following the pattern set by cancel_test.go/run_test.go of
// testing the RunE body as a plain function in addition to going through the
// cobra command.

func TestRunWaitDirectForwardsJobRefOnly(t *testing.T) {
	var gotReq *jobsv0.WaitRequest
	client := &fakeClient{
		waitFunc: func(_ context.Context, req *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			gotReq = req
			return &jobsv0.WaitReply{
				Run: &jobsv0.Run{
					State:    jobsv0.RunStateSucceeded,
					ExitCode: &jobsv0.ExitCode{Value: 0},
				},
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runWait(context.Background(), cli, client, &waitOptions{job: "backup"})

	assert.NilError(t, err)
	assert.Check(t, gotReq != nil, "client.Wait was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.RunRef, ""))
	assert.Check(t, is.Equal(gotReq.Condition, ""))
}

func TestRunWaitExitCodeZeroPrintsZeroAndReturnsNil(t *testing.T) {
	client := &fakeClient{
		waitFunc: func(context.Context, *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			return &jobsv0.WaitReply{
				Run: &jobsv0.Run{
					State:    jobsv0.RunStateSucceeded,
					ExitCode: &jobsv0.ExitCode{Value: 0},
				},
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runWait(context.Background(), cli, client, &waitOptions{job: "backup"})

	assert.NilError(t, err)
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "0\n"))

	var statusErr clipkg.StatusError
	assert.Check(t, !errors.As(err, &statusErr), "runWait must not return cli.StatusError on a zero exit code")
}

func TestRunWaitNonZeroExitCodePrintsCodeAndReturnsStatusError(t *testing.T) {
	client := &fakeClient{
		waitFunc: func(context.Context, *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			return &jobsv0.WaitReply{
				Run: &jobsv0.Run{
					State:    jobsv0.RunStateFailed,
					ExitCode: &jobsv0.ExitCode{Value: 3},
				},
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runWait(context.Background(), cli, client, &waitOptions{job: "backup"})

	assert.Check(t, is.Equal(cli.OutBuffer().String(), "3\n"))

	var statusErr clipkg.StatusError
	assert.Check(t, errors.As(err, &statusErr), "expected a cli.StatusError, got %v", err)
	assert.Check(t, is.Equal(statusErr.StatusCode, 3))
}

func TestRunWaitNoExitCodePrintsNoNumberAndReturnsGenericStatusError(t *testing.T) {
	client := &fakeClient{
		waitFunc: func(context.Context, *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			return &jobsv0.WaitReply{
				Run: &jobsv0.Run{
					State: jobsv0.RunStateFailed,
					Error: "failed to pull image",
				},
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runWait(context.Background(), cli, client, &waitOptions{job: "backup"})

	assert.Check(t, is.Equal(cli.OutBuffer().String(), ""), "no exit code to print when the run never started a container")
	assert.Check(t, strings.Contains(cli.ErrBuffer().String(), "failed to pull image"), "error message should surface Run.Error: %v", cli.ErrBuffer().String())

	var statusErr clipkg.StatusError
	assert.Check(t, errors.As(err, &statusErr), "expected a cli.StatusError, got %v", err)
	assert.Check(t, is.Equal(statusErr.StatusCode, 1))
}

// TestRunWaitNoExitCodeAndNoErrorFallsBackToState covers a run that never
// started a container but also has no Run.Error -- e.g. cancelled while
// still pending (RunStateCancelled's godoc, unlike RunStatePending's, makes
// no promise that Error is set). The message must still be informative
// instead of ending in a bare, unexplained colon.
func TestRunWaitNoExitCodeAndNoErrorFallsBackToState(t *testing.T) {
	client := &fakeClient{
		waitFunc: func(context.Context, *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			return &jobsv0.WaitReply{
				Run: &jobsv0.Run{
					State: jobsv0.RunStateCancelled,
					Error: "",
				},
			}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runWait(context.Background(), cli, client, &waitOptions{job: "backup"})

	assert.Check(t, is.Equal(cli.OutBuffer().String(), ""), "no exit code to print when the run never started a container")
	assert.Check(t, strings.Contains(cli.ErrBuffer().String(), jobsv0.RunStateCancelled), "error message should fall back to Run.State when Run.Error is empty: %v", cli.ErrBuffer().String())

	var statusErr clipkg.StatusError
	assert.Check(t, errors.As(err, &statusErr), "expected a cli.StatusError, got %v", err)
	assert.Check(t, is.Equal(statusErr.StatusCode, 130))
}

func TestNewWaitCommandForwardsRunAndCondition(t *testing.T) {
	var gotReq *jobsv0.WaitRequest
	client := &fakeClient{
		waitFunc: func(_ context.Context, req *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			gotReq = req
			return &jobsv0.WaitReply{Run: &jobsv0.Run{ID: "run-7", State: jobsv0.RunStateRunning}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newWaitCommand(cli, client)
	cmd.SetArgs([]string{"--run", "run-7", "--condition", "running", "backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Wait was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.RunRef, "run-7"))
	assert.Check(t, is.Equal(gotReq.Condition, "running"))
}

// With --condition running, a run that is up satisfies the wait: there is no
// exit code yet, so the run ID is printed (like `docker job run`) and the
// command succeeds.
func TestRunWaitConditionRunningPrintsRunID(t *testing.T) {
	client := &fakeClient{
		waitFunc: func(context.Context, *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			return &jobsv0.WaitReply{Run: &jobsv0.Run{ID: "run-7", State: jobsv0.RunStateRunning}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runWait(context.Background(), cli, client, &waitOptions{job: "backup", condition: jobsv0.WaitConditionRunning})

	assert.NilError(t, err)
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "run-7\n"))
}

// A run stopped by the daemon (timeout) or by Cancel carries the exit code of
// the container's last gasp -- 137, or even 0 when it handles SIGTERM. That
// must not be mistaken for the job's own success or failure: the process
// exits with 124 (timed out) or 130 (cancelled), like `docker compose run`
// does for the same run, following timeout(1) and the SIGINT convention.
func TestRunWaitTimedOutAndCancelledRunsFail(t *testing.T) {
	testCases := []struct {
		doc    string
		state  string
		exit   int64
		status int
	}{
		{doc: "timed out, killed", state: jobsv0.RunStateTimedOut, exit: 137, status: 124},
		{doc: "timed out, clean exit on SIGTERM", state: jobsv0.RunStateTimedOut, exit: 0, status: 124},
		{doc: "cancelled, clean exit on SIGTERM", state: jobsv0.RunStateCancelled, exit: 0, status: 130},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			client := &fakeClient{
				waitFunc: func(context.Context, *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
					return &jobsv0.WaitReply{Run: &jobsv0.Run{State: tc.state, ExitCode: &jobsv0.ExitCode{Value: tc.exit}}}, nil
				},
			}
			cli := test.NewFakeCli(nil)

			err := runWait(context.Background(), cli, client, &waitOptions{job: "backup"})

			assert.Check(t, is.Equal(cli.OutBuffer().String(), ""))
			assert.Check(t, strings.Contains(cli.ErrBuffer().String(), tc.state), "stderr should name the state: %q", cli.ErrBuffer().String())
			var statusErr clipkg.StatusError
			assert.Check(t, errors.As(err, &statusErr), "expected a cli.StatusError, got %v", err)
			assert.Check(t, is.Equal(statusErr.StatusCode, tc.status))
		})
	}
}

// With --condition running, a run that already finished still satisfies the
// wait (it jumped past the condition): it is reported like without the flag,
// with its exit code rather than its ID.
func TestRunWaitConditionRunningOnFinishedRunPrintsExitCode(t *testing.T) {
	client := &fakeClient{
		waitFunc: func(context.Context, *jobsv0.WaitRequest) (*jobsv0.WaitReply, error) {
			return &jobsv0.WaitReply{Run: &jobsv0.Run{ID: "run-7", State: jobsv0.RunStateSucceeded, ExitCode: &jobsv0.ExitCode{Value: 0}}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runWait(context.Background(), cli, client, &waitOptions{job: "backup", condition: jobsv0.WaitConditionRunning})

	assert.NilError(t, err)
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "0\n"))
}
