package job

import (
	"context"
	"errors"
	"testing"

	"github.com/containerd/errdefs"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// jobsv0.Jobs must keep satisfying jobsClient structurally: this line stops
// compiling the day the real interface no longer matches what the job
// package expects from it.
var _ jobsClient = jobsv0.Jobs(nil)

// fakeClient is a test double for jobsClient, following the fakeClient
// pattern used across cli/command/* (see cli/command/volume/client_test.go):
// one <method>Func field per method, a nil field falling back to a harmless
// zero-value response.
type fakeClient struct {
	listFunc     func(ctx context.Context, req *jobsv0.ListRequest) (*jobsv0.ListReply, error)
	inspectFunc  func(ctx context.Context, req *jobsv0.InspectRequest) (*jobsv0.InspectReply, error)
	listRunsFunc func(ctx context.Context, req *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error)
}

func (c *fakeClient) List(ctx context.Context, req *jobsv0.ListRequest) (*jobsv0.ListReply, error) {
	if c.listFunc != nil {
		return c.listFunc(ctx, req)
	}
	return &jobsv0.ListReply{}, nil
}

func (c *fakeClient) Inspect(ctx context.Context, req *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
	if c.inspectFunc != nil {
		return c.inspectFunc(ctx, req)
	}
	return &jobsv0.InspectReply{}, nil
}

func (c *fakeClient) ListRuns(ctx context.Context, req *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
	if c.listRunsFunc != nil {
		return c.listRunsFunc(ctx, req)
	}
	return &jobsv0.ListRunsReply{}, nil
}

var _ jobsClient = (*fakeClient)(nil)

func TestWrapJobsError(t *testing.T) {
	plain := errors.New("boom")

	testCases := []struct {
		doc  string
		err  error
		want string // exact message; empty means the error is returned unchanged
	}{
		{doc: "nil"},
		{doc: "non-status error", err: plain},
		{
			doc:  "unimplemented points at the enable flag",
			err:  status.Error(codes.Unimplemented, "unknown service"),
			want: `the daemon does not support jobs; start dockerd with --extension-enable org.mobyproject.jobs.v0 (or set "extensions.enable" in daemon.json)`,
		},
		{
			doc:  "unavailable is retryable and keeps the daemon message",
			err:  status.Error(codes.Unavailable, "reconciling runs"),
			want: "the jobs API is unavailable (the daemon may be starting or unreachable), retry in a moment: reconciling runs",
		},
		{doc: "not found", err: status.Error(codes.NotFound, `job "backup" not found`), want: `job "backup" not found`},
		{doc: "already exists", err: status.Error(codes.AlreadyExists, "name taken"), want: "name taken"},
		{doc: "failed precondition", err: status.Error(codes.FailedPrecondition, "job is already running"), want: "job is already running"},
		{doc: "invalid argument", err: status.Error(codes.InvalidArgument, "bad runs_removal"), want: "bad runs_removal"},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			got := wrapJobsError(tc.err)

			switch {
			case tc.err == nil:
				assert.NilError(t, got)
			case tc.want == "":
				assert.Check(t, is.Equal(got, tc.err))
			default:
				assert.Error(t, got, tc.want)
			}
		})
	}
}

// A cancelled context (Ctrl-C during a blocking call such as `docker job wait`)
// comes back from gRPC as a Canceled status, which the CLI's main only
// silences when it is recognisable as a cancellation.
func TestWrapJobsErrorCanceledIsRecognisedAsCancellation(t *testing.T) {
	got := wrapJobsError(status.FromContextError(context.Canceled).Err())

	assert.Check(t, errdefs.IsCanceled(got), "got %v", got)
}
