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

// Output convention choice: client.Pause has no reply message (like
// client.Remove), so on success runPause prints jobRef (the job name/ID
// passed as the single positional argument) on dockerCli.Out(), mirroring
// runRm (see rm.go) and `docker volume rm`/`docker network rm`.

func TestNewPauseCommandForwardsJobRef(t *testing.T) {
	var gotReq *jobsv0.PauseRequest
	client := &fakeClient{
		pauseFunc: func(_ context.Context, req *jobsv0.PauseRequest) error {
			gotReq = req
			return nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newPauseCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Pause was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
}

func TestNewPauseCommandPrintsJobOnSuccess(t *testing.T) {
	client := &fakeClient{
		pauseFunc: func(context.Context, *jobsv0.PauseRequest) error {
			return nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newPauseCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(cli.OutBuffer().String(), "backup\n"))
}

func TestNewPauseCommandRequiresExactlyOneArg(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(nil)

	cmd := newPauseCommand(cli, client)
	cmd.SetArgs(nil)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "requires 1 argument")
}

func TestNewPauseCommandUnimplementedIsActionable(t *testing.T) {
	client := &fakeClient{
		pauseFunc: func(context.Context, *jobsv0.PauseRequest) error {
			return status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newPauseCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	// The raw gRPC status dump must not leak to the user.
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewPauseCommandOtherErrorsPassThroughUnwrapped(t *testing.T) {
	errText := `job "backup" not found`
	client := &fakeClient{
		pauseFunc: func(context.Context, *jobsv0.PauseRequest) error {
			return errors.New(errText)
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newPauseCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, errText)
}

// TestRunPauseDirect exercises runPause directly (bypassing cobra arg
// parsing), following the pattern set by cancel_test.go/rm_test.go of
// testing the RunE body as a plain function in addition to going through the
// cobra command.
func TestRunPauseDirect(t *testing.T) {
	var gotReq *jobsv0.PauseRequest
	client := &fakeClient{
		pauseFunc: func(_ context.Context, req *jobsv0.PauseRequest) error {
			gotReq = req
			return nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runPause(context.Background(), cli, client, "backup")

	assert.NilError(t, err)
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "backup\n"))
}

func TestRunPauseDirectOtherErrorsPassThroughUnwrapped(t *testing.T) {
	errText := `job "backup" not found`
	client := &fakeClient{
		pauseFunc: func(context.Context, *jobsv0.PauseRequest) error {
			return errors.New(errText)
		},
	}
	cli := test.NewFakeCli(nil)

	err := runPause(context.Background(), cli, client, "backup")

	assert.ErrorContains(t, err, errText)
	// Nothing should be printed on failure.
	assert.Check(t, is.Equal(cli.OutBuffer().String(), ""))
}
