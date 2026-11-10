// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package job

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/cli/internal/jobsapi/v0/protogen"
	extensionclient "github.com/moby/extensions/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// jobsClient is the subset of the jobsv0.Jobs contract consumed by the
// `docker job` commands. It is intentionally narrow: jobsv0.Jobs (the real
// implementation, obtained through github.com/moby/extensions/client against
// the daemon's jobs extension point) already satisfies it structurally, so no
// adapter type is needed in production code -- see the var _ jobsClient
// assertion in client_test.go. Do not widen it speculatively.
type jobsClient interface {
	List(ctx context.Context, req *jobsv0.ListRequest) (*jobsv0.ListReply, error)

	// Inspect returns a job with its latest run summary.
	Inspect(ctx context.Context, req *jobsv0.InspectRequest) (*jobsv0.InspectReply, error)

	// ListRuns returns a job's runs, most recent first. Used as the
	// fallback when the latest run's container is gone.
	ListRuns(ctx context.Context, req *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error)

	// InspectRun returns a single run.
	InspectRun(ctx context.Context, req *jobsv0.InspectRunRequest) (*jobsv0.InspectRunReply, error)
}

// newJobsClient dials the daemon's jobs extension point and returns a
// jobsClient backed by github.com/moby/extensions/client, plus a closer for
// the underlying connection that callers must invoke once done with it.
//
// Resolve is lazy: a daemon without the feature only surfaces
// codes.Unimplemented on the first RPC call (e.g. the first List), not on
// newJobsClient itself.
func newJobsClient(dockerCli command.Cli) (jobsClient, func() error, error) {
	exts, err := extensionclient.New(dockerCli.Client(), extensionclient.WithGRPCPoint(protogen.ClientPoint))
	if err != nil {
		return nil, nil, err
	}
	client, err := extensionclient.Resolve(exts, jobsv0.Point)
	if err != nil {
		_ = exts.Close()
		return nil, nil, err
	}
	return client, exts.Close, nil
}

// enableJobsFlag is the dockerd option enabling the daemon's jobs extension.
const enableJobsFlag = "--extension-enable org.mobyproject.jobs.v0"

// wrapJobsError turns the gRPC status errors of the Jobs API into actionable
// CLI errors instead of raw `rpc error: code = ...` dumps. Unimplemented means
// the extension is not enabled; Unavailable means the daemon is still starting
// (or the extension failed to activate), or cannot be reached at all, which is
// worth a retry. The other codes the contract defines carry their own message.
// Any other error (including nil) is returned unchanged.
func wrapJobsError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() { //nolint:exhaustive // only the codes defined by the Jobs API contract are mapped
	case codes.Unimplemented:
		return errors.New("the daemon does not support jobs; start dockerd with " + enableJobsFlag + ` (or set "extensions.enable" in daemon.json)`)
	case codes.Canceled:
		// A cancelled context (Ctrl-C) must stay recognisable as such, so that the
		// CLI exits quietly like it does for the other commands.
		return context.Canceled
	case codes.Unavailable:
		return fmt.Errorf("the jobs API is unavailable (the daemon may be starting or unreachable), retry in a moment: %s", st.Message())
	case codes.NotFound, codes.AlreadyExists, codes.FailedPrecondition, codes.InvalidArgument:
		return errors.New(st.Message())
	default:
		return err
	}
}
