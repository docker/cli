package job

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli/streams"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/cli/internal/test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// Flag convention: --label is repeatable (pflag StringArrayVar, not
// StringSliceVar) so a label value containing a comma is not silently split
// -- consistent with jobsv0.PruneRequest.Labels/ListRequest.Labels both being
// "key" or "key=value" filter entries, matched conjunctively (see jobs.go).
//
// Output convention choice: each removed job ID is printed on its own line
// on dockerCli.Out(), mirroring `docker container prune`/`docker volume
// prune` printing one identifier per line for what was removed. When nothing
// was pruned, runPrune prints nothing and returns a nil error -- unlike
// `docker system prune`'s reclaimed-space summary, there is no secondary
// metric worth reporting here (Prune only ever reports removed job IDs), so
// an empty, silent success keeps the command composable/scriptable (e.g.
// `docker job prune --label foo` in a script can just check the exit code).

func TestNewPruneCommandWithoutLabelSendsEmptyRequest(t *testing.T) {
	var gotReq *jobsv0.PruneRequest
	client := &fakeClient{
		pruneFunc: func(_ context.Context, req *jobsv0.PruneRequest) (*jobsv0.PruneReply, error) {
			gotReq = req
			return &jobsv0.PruneReply{}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newPruneCommand(cli, client)
	cmd.SetArgs([]string{"--force"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Prune was not called")
	assert.Check(t, is.Len(gotReq.Labels, 0))
}

func TestNewPruneCommandForwardsRepeatedLabelFlag(t *testing.T) {
	var gotReq *jobsv0.PruneRequest
	client := &fakeClient{
		pruneFunc: func(_ context.Context, req *jobsv0.PruneRequest) (*jobsv0.PruneReply, error) {
			gotReq = req
			return &jobsv0.PruneReply{}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newPruneCommand(cli, client)
	cmd.SetArgs([]string{"--force", "--label", "foo", "--label", "bar=baz"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Prune was not called")
	assert.Check(t, is.DeepEqual(gotReq.Labels, []string{"foo", "bar=baz"}))
}

func TestNewPruneCommandPrintsRemovedJobIDs(t *testing.T) {
	client := &fakeClient{
		pruneFunc: func(context.Context, *jobsv0.PruneRequest) (*jobsv0.PruneReply, error) {
			return &jobsv0.PruneReply{RemovedJobIDs: []string{"job-1", "job-2"}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newPruneCommand(cli, client)
	cmd.SetArgs([]string{"--force"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(cli.OutBuffer().String(), "job-1\njob-2\n"))
}

func TestNewPruneCommandPrintsNothingWhenNoJobsRemoved(t *testing.T) {
	client := &fakeClient{
		pruneFunc: func(context.Context, *jobsv0.PruneRequest) (*jobsv0.PruneReply, error) {
			return &jobsv0.PruneReply{}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newPruneCommand(cli, client)
	cmd.SetArgs([]string{"--force"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(cli.OutBuffer().String(), ""))
}

func TestNewPruneCommandRejectsPositionalArgs(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(nil)

	cmd := newPruneCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "accepts no arguments")
}

func TestNewPruneCommandUnimplementedIsActionable(t *testing.T) {
	client := &fakeClient{
		pruneFunc: func(context.Context, *jobsv0.PruneRequest) (*jobsv0.PruneReply, error) {
			return nil, status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newPruneCommand(cli, client)
	cmd.SetArgs([]string{"--force"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	// The raw gRPC status dump must not leak to the user.
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

// TestRunPruneDirect exercises runPrune directly (bypassing cobra arg
// parsing), following the pattern set by run_test.go/inspect_test.go of
// testing the RunE body as a plain function in addition to going through the
// cobra command.
func TestRunPruneDirect(t *testing.T) {
	var gotReq *jobsv0.PruneRequest
	client := &fakeClient{
		pruneFunc: func(_ context.Context, req *jobsv0.PruneRequest) (*jobsv0.PruneReply, error) {
			gotReq = req
			return &jobsv0.PruneReply{RemovedJobIDs: []string{"job-9"}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runPrune(context.Background(), cli, client, &pruneOptions{labels: []string{"foo"}, force: true})

	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(gotReq.Labels, []string{"foo"}))
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "job-9\n"))
}

func TestNewPruneCommandAsksForConfirmation(t *testing.T) {
	testCases := []struct {
		doc        string
		input      string
		wantPruned bool
	}{
		{doc: "confirmed", input: "y\n", wantPruned: true},
		{doc: "declined", input: "n\n", wantPruned: false},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			var pruned bool
			client := &fakeClient{
				pruneFunc: func(context.Context, *jobsv0.PruneRequest) (*jobsv0.PruneReply, error) {
					pruned = true
					return &jobsv0.PruneReply{RemovedJobIDs: []string{"job-1"}}, nil
				},
			}
			cli := test.NewFakeCli(nil)
			cli.SetIn(streams.NewIn(io.NopCloser(strings.NewReader(tc.input))))

			cmd := newPruneCommand(cli, client)
			cmd.SetArgs(nil)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err := cmd.Execute()

			assert.Check(t, is.Contains(cli.OutBuffer().String(), "WARNING! This will remove all idle manual jobs."))
			assert.Check(t, is.Equal(pruned, tc.wantPruned))
			if tc.wantPruned {
				assert.NilError(t, err)
				assert.Check(t, is.Contains(cli.OutBuffer().String(), "job-1\n"))
			} else {
				assert.Check(t, errdefs.IsCanceled(err), "got %v", err)
			}
		})
	}
}

func TestNewPruneCommandForceSkipsConfirmation(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(nil)
	// No input is provided: a prompt would be declined.

	cmd := newPruneCommand(cli, client)
	cmd.SetArgs([]string{"-f"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	assert.NilError(t, cmd.Execute())
	assert.Check(t, !strings.Contains(cli.OutBuffer().String(), "WARNING!"), "output: %q", cli.OutBuffer())
}
