package job

import (
	"context"
	"encoding/json"
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

// Output format: `docker job inspect` dumps the raw jobsv0.Job (or
// jobsv0.Run, with --run) as indented JSON on dockerCli.Out(), 2-space
// indent, one object -- not wrapped in an array, unlike the generic
// cli/command/inspect package (which serves multi-reference --format/
// template inspection across several docker CLI resources and is not a good
// fit for this single-object, template-less case). Tests assert on decoded
// JSON fields rather than the exact byte layout, to stay robust to
// formatting-only changes.

func TestNewInspectCommandWithoutRunCallsInspectOnly(t *testing.T) {
	var gotReq *jobsv0.InspectRequest
	client := &fakeClient{
		inspectFunc: func(_ context.Context, req *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			gotReq = req
			return &jobsv0.InspectReply{Job: &jobsv0.Job{ID: "job-1", Name: "backup"}}, nil
		},
		inspectRunFunc: func(context.Context, *jobsv0.InspectRunRequest) (*jobsv0.InspectRunReply, error) {
			t.Fatal("client.InspectRun must not be called without --run")
			return nil, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newInspectCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.Inspect was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))

	var got jobsv0.Job
	assert.NilError(t, json.Unmarshal(cli.OutBuffer().Bytes(), &got))
	assert.Check(t, is.Equal(got.ID, "job-1"))
	assert.Check(t, is.Equal(got.Name, "backup"))
}

func TestNewInspectCommandWithRunCallsInspectRunOnly(t *testing.T) {
	var gotReq *jobsv0.InspectRunRequest
	client := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			t.Fatal("client.Inspect must not be called with --run")
			return nil, nil
		},
		inspectRunFunc: func(_ context.Context, req *jobsv0.InspectRunRequest) (*jobsv0.InspectRunReply, error) {
			gotReq = req
			return &jobsv0.InspectRunReply{Run: &jobsv0.Run{ID: "run-1", JobID: "job-1"}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newInspectCommand(cli, client)
	cmd.SetArgs([]string{"--run", "latest", "backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, gotReq != nil, "client.InspectRun was not called")
	assert.Check(t, is.Equal(gotReq.JobRef, "backup"))
	assert.Check(t, is.Equal(gotReq.RunRef, "latest"))

	var got jobsv0.Run
	assert.NilError(t, json.Unmarshal(cli.OutBuffer().Bytes(), &got))
	assert.Check(t, is.Equal(got.ID, "run-1"))
	assert.Check(t, is.Equal(got.JobID, "job-1"))
}

func TestNewInspectCommandWithArbitraryRunRefForwarded(t *testing.T) {
	var gotReq *jobsv0.InspectRunRequest
	client := &fakeClient{
		inspectRunFunc: func(_ context.Context, req *jobsv0.InspectRunRequest) (*jobsv0.InspectRunReply, error) {
			gotReq = req
			return &jobsv0.InspectRunReply{Run: &jobsv0.Run{ID: "run-42"}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newInspectCommand(cli, client)
	cmd.SetArgs([]string{"--run", "run-42", "backup"})
	assert.NilError(t, cmd.Execute())

	assert.Check(t, is.Equal(gotReq.RunRef, "run-42"))
}

func TestNewInspectCommandRequiresExactlyOneArg(t *testing.T) {
	client := &fakeClient{}
	cli := test.NewFakeCli(nil)

	cmd := newInspectCommand(cli, client)
	cmd.SetArgs(nil)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, "requires 1 argument")
}

func TestNewInspectCommandUnimplementedOnInspectIsActionable(t *testing.T) {
	client := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return nil, status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newInspectCommand(cli, client)
	cmd.SetArgs([]string{"backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	// The raw gRPC status dump must not leak to the user.
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewInspectCommandUnimplementedOnInspectRunIsActionable(t *testing.T) {
	client := &fakeClient{
		inspectRunFunc: func(context.Context, *jobsv0.InspectRunRequest) (*jobsv0.InspectRunReply, error) {
			return nil, status.Error(codes.Unimplemented, "unknown service jobsv0.Jobs")
		},
	}
	cli := test.NewFakeCli(nil)

	cmd := newInspectCommand(cli, client)
	cmd.SetArgs([]string{"--run", "latest", "backup"})
	cmd.SetOut(io.Discard)
	err := cmd.Execute()

	assert.ErrorContains(t, err, `--extension-enable org.mobyproject.jobs.v0`)
	assert.Check(t, !strings.Contains(err.Error(), "rpc error"), "error message leaks the raw gRPC status: %v", err)
	assert.Check(t, !strings.Contains(err.Error(), "Unimplemented"), "error message leaks the raw gRPC code: %v", err)
}

func TestNewInspectCommandOtherErrorsPassThroughUnwrapped(t *testing.T) {
	testCases := []struct {
		doc     string
		run     string
		errText string
	}{
		{doc: "job not found, without --run", errText: `job "backup" not found`},
		{doc: "run not found, with --run", run: "latest", errText: `run "latest" not found for job "backup"`},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			client := &fakeClient{
				inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
					return nil, errors.New(tc.errText)
				},
				inspectRunFunc: func(context.Context, *jobsv0.InspectRunRequest) (*jobsv0.InspectRunReply, error) {
					return nil, errors.New(tc.errText)
				},
			}
			cli := test.NewFakeCli(nil)

			cmd := newInspectCommand(cli, client)
			args := []string{"backup"}
			if tc.run != "" {
				args = append([]string{"--run", tc.run}, args...)
			}
			cmd.SetArgs(args)
			cmd.SetOut(io.Discard)
			err := cmd.Execute()

			assert.ErrorContains(t, err, tc.errText)
		})
	}
}

// TestRunInspectDirect exercises runInspect directly (bypassing cobra arg
// parsing), following the pattern set by run_test.go/logs_test.go of testing
// the RunE body as a plain function in addition to going through the cobra
// command.
func TestRunInspectDirect(t *testing.T) {
	client := &fakeClient{
		inspectFunc: func(_ context.Context, req *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return &jobsv0.InspectReply{Job: &jobsv0.Job{ID: "job-1", Name: req.JobRef}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runInspect(context.Background(), cli, client, &inspectOptions{job: "backup"})

	assert.NilError(t, err)
	var got jobsv0.Job
	assert.NilError(t, json.Unmarshal(cli.OutBuffer().Bytes(), &got))
	assert.Check(t, is.Equal(got.Name, "backup"))
}

func TestRunInspectDirectWithRun(t *testing.T) {
	client := &fakeClient{
		inspectRunFunc: func(_ context.Context, req *jobsv0.InspectRunRequest) (*jobsv0.InspectRunReply, error) {
			return &jobsv0.InspectRunReply{Run: &jobsv0.Run{ID: "run-1", JobID: req.JobRef}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runInspect(context.Background(), cli, client, &inspectOptions{job: "backup", run: "latest"})

	assert.NilError(t, err)
	var got jobsv0.Run
	assert.NilError(t, json.Unmarshal(cli.OutBuffer().Bytes(), &got))
	assert.Check(t, is.Equal(got.ID, "run-1"))
	assert.Check(t, is.Equal(got.JobID, "backup"))
}

// The container spec is a JSON document carried as bytes: it must be printed
// as that document, not as the base64 string a []byte marshals to.
func TestRunInspectPrintsContainerSpecAsJSON(t *testing.T) {
	client := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return &jobsv0.InspectReply{Job: &jobsv0.Job{
				ID:   "job-1",
				Name: "backup",
				Spec: &jobsv0.JobSpec{
					ContainerSpec:  []byte(`{"Image":"busybox","Cmd":["echo","hi"]}`),
					TimeoutSeconds: 30,
				},
			}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runInspect(context.Background(), cli, client, &inspectOptions{job: "backup"})

	assert.NilError(t, err)
	var got struct {
		Name string
		Spec struct {
			ContainerSpec  map[string]any
			TimeoutSeconds int64
		}
	}
	assert.NilError(t, json.Unmarshal(cli.OutBuffer().Bytes(), &got), "output: %s", cli.OutBuffer())
	assert.Check(t, is.Equal(got.Name, "backup"))
	assert.Check(t, is.Equal(got.Spec.ContainerSpec["Image"], "busybox"))
	assert.Check(t, is.Equal(got.Spec.TimeoutSeconds, int64(30)), "the other spec fields must be kept")
}

func TestRunInspectJobWithoutSpec(t *testing.T) {
	client := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return &jobsv0.InspectReply{Job: &jobsv0.Job{ID: "job-1", Name: "backup"}}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runInspect(context.Background(), cli, client, &inspectOptions{job: "backup"})

	assert.NilError(t, err)
	assert.Check(t, strings.Contains(cli.OutBuffer().String(), `"Spec": null`), "output: %s", cli.OutBuffer())
}

// A reply without a job used to be printed as null. Building the view of the
// job must not turn that into a panic.
func TestRunInspectReplyWithoutJob(t *testing.T) {
	client := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			return &jobsv0.InspectReply{}, nil
		},
	}
	cli := test.NewFakeCli(nil)

	err := runInspect(context.Background(), cli, client, &inspectOptions{job: "backup"})

	assert.NilError(t, err)
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "null\n"))
}
