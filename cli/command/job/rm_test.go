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

// --runs is forwarded to jobsv0.RemoveRequest.RunsRemoval as a raw string,
// with no client-side mapping/enum: an unset flag sends an empty string,
// which the daemon defaults to "keep" (see jobsv0.RunsKeep and
// jobsv0.RemoveRequest.RunsRemoval in jobs.go). Validation of the value is
// left entirely to the daemon.
//
// Output convention choice: on success runRm prints opts.job (the job
// name/ID passed as the single positional argument) on dockerCli.Out(),
// mirroring `docker volume rm`/`docker network rm` (see
// cli/command/volume/remove.go and cli/command/network/remove.go), which
// both print the resource identifier as given by the caller via
// fmt.Fprintln(dockerCli.Out(), name) on a successful removal.

func TestNewRmCommandForwardsJobRefWithoutRunsFlag(t *testing.T) {
	var gotReq *jobsv0.RemoveRequest
	client := &fakeClient{
		removeFunc: func(_ context.Context, req *jobsv0.RemoveRequest) error {
			gotReq = req
			return nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRmCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Remove was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.RunsRemoval, ""))
}

func TestNewRmCommandForwardsRunsFlag(t *testing.T) {
	var gotReq *jobsv0.RemoveRequest
	client := &fakeClient{
		removeFunc: func(_ context.Context, req *jobsv0.RemoveRequest) error {
			gotReq = req
			return nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRmCommand(cli, client)
	cmd.SetArgs([]string{"--runs", "remove", "backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Remove was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.RunsRemoval, "remove"))
}

func TestNewRmCommandForwardsRunsFlagRawValue(t *testing.T) {
	// The CLI performs no validation/mapping of --runs: an arbitrary string
	// is forwarded as-is, leaving the daemon to reject invalid values.
	var gotReq *jobsv0.RemoveRequest
	client := &fakeClient{
		removeFunc: func(_ context.Context, req *jobsv0.RemoveRequest) error {
			gotReq = req
			return nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRmCommand(cli, client)
	cmd.SetArgs([]string{"--runs", "bogus-value", "backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Remove was not called")
	assert.Check(t, is.Equal(gotReq.RunsRemoval, "bogus-value"))
}

func TestNewRmCommandPrintsJobOnSuccess(t *testing.T) {
	client := &fakeClient{
		removeFunc: func(context.Context, *jobsv0.RemoveRequest) error {
			return nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRmCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(cli.OutBuffer().String(), "backup\n"))
}

func TestNewRmCommandRequiresExactlyOneArg(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(nil)

	cmd := newRmCommand(cli, client)
	cmd.SetArgs(nil)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "requires 1 argument")
}

func TestNewRmCommandUnimplementedIsActionable(t *testing.T) {
	client := &fakeClient{
		removeFunc: func(context.Context, *jobsv0.RemoveRequest) error {
			return status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRmCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	// The raw gRPC status dump must not leak to the user.
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewRmCommandOtherErrorsPassThroughUnwrapped(t *testing.T) {
	errText := `job "backup" not found`
	client := &fakeClient{
		removeFunc: func(context.Context, *jobsv0.RemoveRequest) error {
			return errors.New(errText)
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newRmCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, errText)
}

// TestRunRmDirect exercises runRm directly (bypassing cobra arg parsing),
// following the pattern set by cancel_test.go/prune_test.go of testing the
// RunE body as a plain function in addition to going through the cobra
// command.
func TestRunRmDirect(t *testing.T) {
	var gotReq *jobsv0.RemoveRequest
	client := &fakeClient{
		removeFunc: func(_ context.Context, req *jobsv0.RemoveRequest) error {
			gotReq = req
			return nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runRm(context.Background(), cli, client, &rmOptions{job: "backup", runs: jobsv0.RunsRemoveFinished})

	assert.NilError(t, err)
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.RunsRemoval, jobsv0.RunsRemoveFinished))
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "backup\n"))
}

func TestRunRmDirectOtherErrorsPassThroughUnwrapped(t *testing.T) {
	errText := `job "backup" not found`
	client := &fakeClient{
		removeFunc: func(context.Context, *jobsv0.RemoveRequest) error {
			return errors.New(errText)
		},
	}
	cli := test.NewFakeCli(nil)

	err := runRm(context.Background(), cli, client, &rmOptions{job: "backup"})

	assert.ErrorContains(t, err, errText)
	// Nothing should be printed on failure.
	assert.Check(t, is.Equal(cli.OutBuffer().String(), ""))
}

// `docker container rm`, `volume rm` and `network rm` are all reachable as
// `remove`: keep the same spelling.
func TestNewRmCommandHasRemoveAlias(t *testing.T) {
	cmd := newRmCommand(test.NewFakeCli(nil), &fakeClient{})

	assert.Check(t, is.Contains(cmd.Aliases, "remove"))
}
