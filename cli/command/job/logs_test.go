package job

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	dockerclient "github.com/moby/moby/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// fakeDockerAPIClient is a test double for the subset of client.APIClient
// consumed by `docker job logs` (ContainerInspect, ContainerLogs), following
// the same embedding pattern as cli/command/container's fakeClient: embed
// the zero-value concrete dockerclient.Client (which satisfies
// client.APIClient) and override only what the command under test calls.
type fakeDockerAPIClient struct {
	dockerclient.Client
	inspectFunc func(containerID string) (dockerclient.ContainerInspectResult, error)
	logFunc     func(containerID string, opts dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error)
}

func (f *fakeDockerAPIClient) ContainerInspect(_ context.Context, containerID string, _ dockerclient.ContainerInspectOptions) (dockerclient.ContainerInspectResult, error) {
	if f.inspectFunc != nil {
		return f.inspectFunc(containerID)
	}
	return dockerclient.ContainerInspectResult{}, nil
}

func (f *fakeDockerAPIClient) ContainerLogs(_ context.Context, containerID string, opts dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
	if f.logFunc != nil {
		return f.logFunc(containerID, opts)
	}
	return mockJobContainerLogsResult(""), nil
}

func mockJobContainerLogsResult(content string) dockerclient.ContainerLogsResult {
	return io.NopCloser(strings.NewReader(content))
}

// ttyInspectResult is a ContainerInspect result for a TTY container, the
// only demux path exercised here -- see the FIXME in
// cli/command/container/logs_test.go about mocking the non-TTY stdcopy
// stream; this test suite follows the same simplification.
func ttyInspectResult() dockerclient.ContainerInspectResult {
	return dockerclient.ContainerInspectResult{
		Container: container.InspectResponse{
			Config: &container.Config{Tty: true},
		},
	}
}

func TestNewLogsCommandFollowFlagForwarded(t *testing.T) {
	testCases := []struct {
		doc        string
		args       []string
		wantFollow bool
	}{
		{doc: "long flag", args: []string{"backup", "--follow"}, wantFollow: true},
		{doc: "short flag", args: []string{"backup", "-f"}, wantFollow: true},
		{doc: "flag omitted defaults to false", args: []string{"backup"}, wantFollow: false},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			var gotOpts dockerclient.ContainerLogsOptions
			jobs := &fakeClient{
				inspectFunc: func(_ context.Context, req *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
					assert.Check(t, is.Equal(req.JobRef, "backup"))
					return &jobsv0.InspectReply{Job: &jobsv0.Job{
						Name:      "backup",
						LatestRun: &jobsv0.Run{ID: "run-1", ContainerID: "abc123"},
					}}, nil
				},
			}
			docker := &fakeDockerAPIClient{
				inspectFunc: func(string) (dockerclient.ContainerInspectResult, error) { return ttyInspectResult(), nil },
				logFunc: func(containerID string, opts dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
					gotOpts = opts
					return mockJobContainerLogsResult("[backup] done"), nil
				},
			}
			cli := test.NewFakeCli(docker)

			// A follow=true run only returns once its context is done: after
			// streaming the current run to completion, `logs -f` waits for
			// the job's next run to appear (see waitForNextRun) rather than
			// returning, exactly like `docker logs -f` waits on a live
			// container. This test only cares that the flag reaches
			// ContainerLogsOptions, which happens on the first attach, so a
			// short timeout stands in for the Ctrl-C that would normally end
			// a real `-f` session.
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			cmd := newLogsCommand(cli, jobs)
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			assert.NilError(t, cmd.ExecuteContext(ctx))

			assert.Check(t, is.Equal(gotOpts.Follow, tc.wantFollow))
		})
	}
}

func TestNewLogsCommandResolvesContainerIDFromLatestRun(t *testing.T) {
	jobs := &fakeClient{
		inspectFunc: func(_ context.Context, req *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return &jobsv0.InspectReply{Job: &jobsv0.Job{
				Name:      "backup",
				LatestRun: &jobsv0.Run{ID: "run-1", ContainerID: "abc123", ContainerGone: false},
			}}, nil
		},
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			t.Fatal("ListRuns must not be called when LatestRun's container is not gone")
			return nil, nil
		},
	}
	var gotContainerID string
	docker := &fakeDockerAPIClient{
		inspectFunc: func(containerID string) (dockerclient.ContainerInspectResult, error) {
			gotContainerID = containerID
			return ttyInspectResult(), nil
		},
		logFunc: func(containerID string, _ dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			return mockJobContainerLogsResult("[backup] done"), nil
		},
	}
	cli := test.NewFakeCli(docker)

	cmd := newLogsCommand(cli, jobs)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(gotContainerID, "abc123"))
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "[backup] done"))
}

func TestNewLogsCommandFallsBackWhenContainerGone(t *testing.T) {
	jobs := &fakeClient{
		inspectFunc: func(_ context.Context, req *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return &jobsv0.InspectReply{Job: &jobsv0.Job{
				Name:      "backup",
				LatestRun: &jobsv0.Run{ID: "run-2", ContainerGone: true},
			}}, nil
		},
		listRunsFunc: func(_ context.Context, req *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			assert.Check(t, is.Equal(req.JobRef, "backup"))
			return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{
				{ID: "run-2", ContainerGone: true},
				{ID: "run-1", ContainerID: "def456", ContainerGone: false},
			}}, nil
		},
	}
	var gotContainerID string
	docker := &fakeDockerAPIClient{
		inspectFunc: func(containerID string) (dockerclient.ContainerInspectResult, error) {
			gotContainerID = containerID
			return ttyInspectResult(), nil
		},
		logFunc: func(containerID string, _ dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			return mockJobContainerLogsResult("[backup] done"), nil
		},
	}
	cli := test.NewFakeCli(docker)

	cmd := newLogsCommand(cli, jobs)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(gotContainerID, "def456"))
}

func TestNewLogsCommandNeverRunError(t *testing.T) {
	jobs := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return &jobsv0.InspectReply{Job: &jobsv0.Job{Name: "migrate", LatestRun: nil}}, nil
		},
	}
	docker := &fakeDockerAPIClient{
		logFunc: func(string, dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			t.Fatal("ContainerLogs must not be called when the job never ran")
			return nil, nil
		},
	}
	cli := test.NewFakeCli(docker)

	cmd := newLogsCommand(cli, jobs)
	cmd.SetArgs([]string{"migrate"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `job "migrate" has never run`)
}

func TestNewLogsCommandNoLogsAvailableError(t *testing.T) {
	jobs := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return &jobsv0.InspectReply{Job: &jobsv0.Job{
				Name:      "backup",
				LatestRun: &jobsv0.Run{ID: "run-1", ContainerGone: true},
			}}, nil
		},
		listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
			return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{{ID: "run-1", ContainerGone: true}}}, nil
		},
	}
	docker := &fakeDockerAPIClient{
		logFunc: func(string, dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			t.Fatal("ContainerLogs must not be called when no run has a container")
			return nil, nil
		},
	}
	cli := test.NewFakeCli(docker)

	cmd := newLogsCommand(cli, jobs)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `no logs available for job "backup"`)
}

func TestNewLogsCommandUnimplementedIsActionable(t *testing.T) {
	jobs := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return nil, status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(&fakeDockerAPIClient{})

	cmd := newLogsCommand(cli, jobs)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewLogsCommandStreamsContainerOutput(t *testing.T) {
	jobs := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return &jobsv0.InspectReply{Job: &jobsv0.Job{
				Name:      "backup",
				LatestRun: &jobsv0.Run{ID: "run-1", ContainerID: "abc123"},
			}}, nil
		},
	}
	docker := &fakeDockerAPIClient{
		inspectFunc: func(string) (dockerclient.ContainerInspectResult, error) { return ttyInspectResult(), nil },
		logFunc: func(string, dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			return mockJobContainerLogsResult("[backup] done\n"), nil
		},
	}
	cli := test.NewFakeCli(docker)

	cmd := newLogsCommand(cli, jobs)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(cli.OutBuffer().String(), "[backup] done\n"))
	assert.Check(t, is.Equal(cli.ErrBuffer().String(), ""))
}

func TestNewLogsCommandNotesFallbackToOlderRun(t *testing.T) {
	testCases := []struct {
		doc        string
		latest     jobsv0.Run
		wantNoteOn []string // substrings expected on stderr; none means stderr stays empty
	}{
		{doc: "latest run has logs", latest: jobsv0.Run{ID: "run-2", ContainerID: "def456"}},
		{doc: "latest run's container is gone", latest: jobsv0.Run{ID: "run-2", ContainerGone: true}, wantNoteOn: []string{"run-1", "run-2"}},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			jobs := &fakeClient{
				inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
					return &jobsv0.InspectReply{Job: &jobsv0.Job{Name: "backup", LatestRun: &tc.latest}}, nil
				},
				listRunsFunc: func(context.Context, *jobsv0.ListRunsRequest) (*jobsv0.ListRunsReply, error) {
					return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{tc.latest, {ID: "run-1", ContainerID: "abc123"}}}, nil
				},
			}
			docker := &fakeDockerAPIClient{
				inspectFunc: func(string) (dockerclient.ContainerInspectResult, error) { return ttyInspectResult(), nil },
				logFunc: func(string, dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
					return mockJobContainerLogsResult("[backup] done"), nil
				},
			}
			cli := test.NewFakeCli(docker)

			cmd := newLogsCommand(cli, jobs)
			cmd.SetArgs([]string{"backup"})
			assert.NilError(t, cmd.Execute())

			assert.Check(t, is.Equal(cli.OutBuffer().String(), "[backup] done"), "the note must not pollute stdout")
			if len(tc.wantNoteOn) == 0 {
				assert.Check(t, is.Equal(cli.ErrBuffer().String(), ""))
			}
			for _, want := range tc.wantNoteOn {
				assert.Check(t, is.Contains(cli.ErrBuffer().String(), want))
			}
		})
	}
}
