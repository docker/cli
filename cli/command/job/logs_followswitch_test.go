package job

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/containerd/errdefs"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/docker/cli/internal/test"
	dockerclient "github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// This file covers the `docker job logs -f` follow-switch behavior: once the
// container backing the followed run exits (ContainerLogs' stream ends with
// a clean EOF), `docker job logs -f` on a schedule job polls the job until a
// new run appears and re-attaches to it, instead of exiting.
//
// waitForNextRun takes an injectable sleep/inspect so its tests run in
// milliseconds with zero real time.Sleep.

// -----------------------------------------------------------------------
// streamRunLogs
// -----------------------------------------------------------------------

func TestStreamRunLogsWritesTTYOutputToStdout(t *testing.T) {
	docker := &fakeDockerAPIClient{
		inspectFunc: func(containerID string) (dockerclient.ContainerInspectResult, error) {
			assert.Check(t, is.Equal(containerID, "abc123"))
			return ttyInspectResult(), nil
		},
		logFunc: func(containerID string, _ dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			assert.Check(t, is.Equal(containerID, "abc123"))
			return mockJobContainerLogsResult("hello\n"), nil
		},
	}
	cli := test.NewFakeCli(docker)

	err := streamRunLogs(context.Background(), cli, "abc123", false)

	assert.NilError(t, err)
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "hello\n"))
}

func TestStreamRunLogsForwardsFollowFlag(t *testing.T) {
	var gotFollow bool
	docker := &fakeDockerAPIClient{
		inspectFunc: func(string) (dockerclient.ContainerInspectResult, error) { return ttyInspectResult(), nil },
		logFunc: func(_ string, opts dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			gotFollow = opts.Follow
			return mockJobContainerLogsResult(""), nil
		},
	}
	cli := test.NewFakeCli(docker)

	assert.NilError(t, streamRunLogs(context.Background(), cli, "abc123", true))
	assert.Check(t, gotFollow)
}

func TestStreamRunLogsPropagatesContainerInspectError(t *testing.T) {
	wantErr := errors.New("boom")
	docker := &fakeDockerAPIClient{
		inspectFunc: func(string) (dockerclient.ContainerInspectResult, error) {
			return dockerclient.ContainerInspectResult{}, wantErr
		},
		logFunc: func(string, dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			t.Fatal("ContainerLogs must not be called when ContainerInspect fails")
			return nil, nil
		},
	}
	cli := test.NewFakeCli(docker)

	err := streamRunLogs(context.Background(), cli, "abc123", false)

	assert.ErrorIs(t, err, wantErr)
}

func TestStreamRunLogsPropagatesContainerLogsError(t *testing.T) {
	wantErr := errors.New("boom")
	docker := &fakeDockerAPIClient{
		inspectFunc: func(string) (dockerclient.ContainerInspectResult, error) { return ttyInspectResult(), nil },
		logFunc: func(string, dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			return nil, wantErr
		},
	}
	cli := test.NewFakeCli(docker)

	err := streamRunLogs(context.Background(), cli, "abc123", false)

	assert.ErrorIs(t, err, wantErr)
}

// -----------------------------------------------------------------------
// waitForNextRun
// -----------------------------------------------------------------------

func TestWaitForNextRunReturnsImmediatelyWhenRunAlreadyChanged(t *testing.T) {
	inspectCalls := 0
	inspect := func() (*jobsv0.Job, error) {
		inspectCalls++
		return &jobsv0.Job{LatestRun: &jobsv0.Run{ID: "run-2", ContainerID: "c2"}}, nil
	}
	sleepCalls := 0
	sleep := func(context.Context) error { sleepCalls++; return nil }

	run, err := waitForNextRun(context.Background(), "run-1", inspect, sleep)

	assert.NilError(t, err)
	assert.Check(t, is.Equal(run.ID, "run-2"))
	assert.Check(t, is.Equal(inspectCalls, 1))
	assert.Check(t, is.Equal(sleepCalls, 0), "sleep must not be called when the run already changed on the first poll")
}

func TestWaitForNextRunPollsUntilRunChanges(t *testing.T) {
	jobs := []*jobsv0.Job{
		{LatestRun: &jobsv0.Run{ID: "run-1"}},
		{LatestRun: &jobsv0.Run{ID: "run-1"}},
		{LatestRun: &jobsv0.Run{ID: "run-2", ContainerID: "c2"}},
	}
	inspectCalls := 0
	inspect := func() (*jobsv0.Job, error) {
		job := jobs[inspectCalls]
		inspectCalls++
		return job, nil
	}
	sleepCalls := 0
	sleep := func(context.Context) error { sleepCalls++; return nil }

	run, err := waitForNextRun(context.Background(), "run-1", inspect, sleep)

	assert.NilError(t, err)
	assert.Check(t, is.Equal(run.ID, "run-2"))
	assert.Check(t, is.Equal(inspectCalls, 3))
	assert.Check(t, is.Equal(sleepCalls, 2))
}

func TestWaitForNextRunReturnsImmediatelyWhenContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	inspectCalled := false
	inspect := func() (*jobsv0.Job, error) {
		inspectCalled = true
		return &jobsv0.Job{}, nil
	}
	sleep := func(context.Context) error {
		t.Fatal("sleep must not be called when the context is already cancelled")
		return nil
	}

	run, err := waitForNextRun(ctx, "run-1", inspect, sleep)

	assert.Check(t, is.Nil(run))
	assert.ErrorIs(t, err, context.Canceled)
	assert.Check(t, !inspectCalled, "inspect must not be called when the context is already cancelled")
}

func TestWaitForNextRunPropagatesCancellationDuringSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inspect := func() (*jobsv0.Job, error) {
		return &jobsv0.Job{LatestRun: &jobsv0.Run{ID: "run-1"}}, nil
	}
	sleep := func(ctx context.Context) error {
		cancel()
		return ctx.Err()
	}

	run, err := waitForNextRun(ctx, "run-1", inspect, sleep)

	assert.Check(t, is.Nil(run))
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWaitForNextRunPropagatesInspectError(t *testing.T) {
	wantErr := errors.New("connection refused")
	inspect := func() (*jobsv0.Job, error) { return nil, wantErr }
	sleep := func(context.Context) error {
		t.Fatal("sleep must not be called when inspect fails")
		return nil
	}

	run, err := waitForNextRun(context.Background(), "run-1", inspect, sleep)

	assert.Check(t, is.Nil(run))
	assert.ErrorIs(t, err, wantErr)
}

func TestWaitForNextRunKeepsPollingWhenLatestRunIsNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inspectCalls := 0
	inspect := func() (*jobsv0.Job, error) {
		inspectCalls++
		// A job that never ran again yet (e.g. a manually-triggered job)
		// reports no LatestRun; polling must not panic on the nil deref
		// and must keep going.
		return &jobsv0.Job{LatestRun: nil}, nil
	}
	sleepCalls := 0
	sleep := func(ctx context.Context) error {
		sleepCalls++
		if sleepCalls >= 3 {
			cancel()
			return ctx.Err()
		}
		return nil
	}

	run, err := waitForNextRun(ctx, "run-1", inspect, sleep)

	assert.Check(t, is.Nil(run))
	assert.ErrorIs(t, err, context.Canceled)
	assert.Check(t, inspectCalls >= 3, "expected polling to continue while LatestRun is nil, got %d inspect calls", inspectCalls)
}

// -----------------------------------------------------------------------
// runLogs: follow-switch integration
// -----------------------------------------------------------------------

func TestRunLogsWithoutFollowDoesNotPollForNextRun(t *testing.T) {
	var inspectCalls int32
	jobs := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			atomic.AddInt32(&inspectCalls, 1)
			return &jobsv0.InspectReply{Job: &jobsv0.Job{
				Name:      "backup",
				LatestRun: &jobsv0.Run{ID: "run-1", ContainerID: "c1"},
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

	err := runLogs(context.Background(), cli, jobs, &logsOptions{job: "backup", follow: false})

	assert.NilError(t, err)
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "[backup] done\n"))
	assert.Check(t, is.Equal(atomic.LoadInt32(&inspectCalls), int32(1)))
}

// TestRunLogsFollowSwitchesToNextRunOnContainerStop drives two full cron
// occurrences through runLogs(follow=true): run-1's container exits (clean
// ContainerLogs EOF), runLogs polls the job and re-attaches to run-2 as soon
// as it appears, then run-2's container also exits and the context is
// cancelled (simulating Ctrl-C) so the otherwise-infinite follow loop
// terminates.
func TestRunLogsFollowSwitchesToNextRunOnContainerStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var inspectCalls int32
	jobs := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			switch atomic.AddInt32(&inspectCalls, 1) {
			case 1:
				// Initial resolution in runLogs, before the first attach.
				return &jobsv0.InspectReply{Job: &jobsv0.Job{
					Name:      "backup",
					Spec:      scheduledSpec(),
					LatestRun: &jobsv0.Run{ID: "run-1", ContainerID: "c1"},
				}}, nil
			case 2:
				// First waitForNextRun poll, after run-1's container exits:
				// run-2 already exists, switch immediately.
				return &jobsv0.InspectReply{Job: &jobsv0.Job{
					Name:      "backup",
					Spec:      scheduledSpec(),
					LatestRun: &jobsv0.Run{ID: "run-2", ContainerID: "c2"},
				}}, nil
			default:
				// Second waitForNextRun poll, after run-2's container
				// exits: no new run yet. Cancel the context here to end
				// the test the same way a Ctrl-C would.
				cancel()
				return &jobsv0.InspectReply{Job: &jobsv0.Job{
					Name:      "backup",
					Spec:      scheduledSpec(),
					LatestRun: &jobsv0.Run{ID: "run-2", ContainerID: "c2"},
				}}, nil
			}
		},
	}
	docker := &fakeDockerAPIClient{
		inspectFunc: func(string) (dockerclient.ContainerInspectResult, error) { return ttyInspectResult(), nil },
		logFunc: func(containerID string, opts dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			assert.Check(t, opts.Follow, "runLogs must request Follow on the docker API when opts.follow is set")
			switch containerID {
			case "c1":
				return mockJobContainerLogsResult("[backup] run-1 done\n"), nil
			case "c2":
				return mockJobContainerLogsResult("[backup] run-2 done\n"), nil
			default:
				t.Fatalf("unexpected containerID %q", containerID)
				return nil, nil
			}
		},
	}
	cli := test.NewFakeCli(docker)

	err := runLogs(ctx, cli, jobs, &logsOptions{job: "backup", follow: true})

	assert.NilError(t, err, "runLogs must exit cleanly (nil error) when the context is cancelled while waiting for the next run")
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "[backup] run-1 done\n[backup] run-2 done\n"))
	assert.Check(t, is.Equal(atomic.LoadInt32(&inspectCalls), int32(3)))
}

// TestRunLogsFollowSkipsRunsWhoseContainerIsAlreadyGone covers the case where
// waitForNextRun picks up a new run whose container has already been removed
// (e.g. a fast job cleaned up right after a successful exit -- observed for
// real against a daemon where Compose removes successful run containers).
// runLogs must keep polling for the run after it instead of trying (and
// failing) to attach to the gone container.
func TestRunLogsFollowSkipsRunsWhoseContainerIsAlreadyGone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var inspectCalls int32
	jobs := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			switch atomic.AddInt32(&inspectCalls, 1) {
			case 1:
				// Initial resolution in runLogs, before the first attach.
				return &jobsv0.InspectReply{Job: &jobsv0.Job{
					Name:      "backup",
					Spec:      scheduledSpec(),
					LatestRun: &jobsv0.Run{ID: "run-1", ContainerID: "c1"},
				}}, nil
			case 2:
				// First waitForNextRun poll: run-2 exists but its container
				// is already gone -- runLogs must NOT attach to it, and
				// must NOT surface an error either; it keeps polling.
				return &jobsv0.InspectReply{Job: &jobsv0.Job{
					Name:      "backup",
					Spec:      scheduledSpec(),
					LatestRun: &jobsv0.Run{ID: "run-2", ContainerID: "should-not-be-attached", ContainerGone: true},
				}}, nil
			case 3:
				// Second poll: run-3 has a live container, switch to it.
				return &jobsv0.InspectReply{Job: &jobsv0.Job{
					Name:      "backup",
					Spec:      scheduledSpec(),
					LatestRun: &jobsv0.Run{ID: "run-3", ContainerID: "c3"},
				}}, nil
			default:
				// Third poll, after run-3's container exits: no new run
				// yet. Cancel the context here to end the test the same
				// way a Ctrl-C would.
				cancel()
				return &jobsv0.InspectReply{Job: &jobsv0.Job{
					Name:      "backup",
					Spec:      scheduledSpec(),
					LatestRun: &jobsv0.Run{ID: "run-3", ContainerID: "c3"},
				}}, nil
			}
		},
	}
	docker := &fakeDockerAPIClient{
		inspectFunc: func(containerID string) (dockerclient.ContainerInspectResult, error) {
			if containerID == "should-not-be-attached" {
				t.Fatalf("ContainerInspect must not be called for a run whose container is already gone")
			}
			return ttyInspectResult(), nil
		},
		logFunc: func(containerID string, opts dockerclient.ContainerLogsOptions) (dockerclient.ContainerLogsResult, error) {
			if containerID == "should-not-be-attached" {
				t.Fatalf("ContainerLogs must not be called for a run whose container is already gone")
			}
			switch containerID {
			case "c1":
				return mockJobContainerLogsResult("[backup] run-1 done\n"), nil
			case "c3":
				return mockJobContainerLogsResult("[backup] run-3 done\n"), nil
			default:
				t.Fatalf("unexpected containerID %q", containerID)
				return nil, nil
			}
		},
	}
	cli := test.NewFakeCli(docker)

	err := runLogs(ctx, cli, jobs, &logsOptions{job: "backup", follow: true})

	assert.NilError(t, err, "runLogs must exit cleanly (nil error) when the context is cancelled while waiting for the next run")
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "[backup] run-1 done\n[backup] run-3 done\n"))
	assert.Check(t, is.Equal(atomic.LoadInt32(&inspectCalls), int32(4)))
}

// scheduledSpec is the spec of a job with a cron schedule, the only kind of
// job `logs -f` can follow past the end of a run.
func scheduledSpec() *jobsv0.JobSpec {
	return &jobsv0.JobSpec{Trigger: &jobsv0.Trigger{Schedule: &jobsv0.ScheduleTrigger{Cron: "* * * * *"}}}
}

// A manual job only runs on an explicit `docker job run`: once its run's
// container exits, `logs -f` has nothing left to follow and must return
// instead of polling forever.
func TestRunLogsFollowReturnsAfterStreamForManualJob(t *testing.T) {
	var inspectCalls int32
	jobs := &fakeClient{
		inspectFunc: func(context.Context, *jobsv0.InspectRequest) (*jobsv0.InspectReply, error) {
			atomic.AddInt32(&inspectCalls, 1)
			return &jobsv0.InspectReply{Job: &jobsv0.Job{
				Name:      "backup",
				Spec:      &jobsv0.JobSpec{Trigger: &jobsv0.Trigger{Manual: true}},
				LatestRun: &jobsv0.Run{ID: "run-1", ContainerID: "c1"},
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

	err := runLogs(context.Background(), cli, jobs, &logsOptions{job: "backup", follow: true})

	assert.NilError(t, err)
	assert.Check(t, is.Equal(cli.OutBuffer().String(), "[backup] done\n"))
	assert.Check(t, is.Equal(atomic.LoadInt32(&inspectCalls), int32(1)), "a manual job must not be polled for a next run")
}

// A new run is visible as soon as its record is written, before its container
// exists. Attaching then would fail on an empty container ID, so the next run
// is only picked up once its container does exist.
func TestWaitForNextRunWaitsForTheRunContainer(t *testing.T) {
	polls := []*jobsv0.Run{
		{ID: "run-2", State: jobsv0.RunStatePending},
		{ID: "run-2", State: jobsv0.RunStateRunning, ContainerID: "c2"},
	}
	i := 0
	inspect := func() (*jobsv0.Job, error) {
		run := polls[i]
		i++
		return &jobsv0.Job{LatestRun: run}, nil
	}
	sleepCalls := 0
	sleep := func(context.Context) error { sleepCalls++; return nil }

	run, err := waitForNextRun(context.Background(), "run-1", inspect, sleep)

	assert.NilError(t, err)
	assert.Check(t, is.Equal(run.ID, "run-2"))
	assert.Check(t, is.Equal(run.ContainerID, "c2"))
	assert.Check(t, is.Equal(sleepCalls, 1))
}

func TestStreamRunLogsReportsRemovedContainerAsUnavailableLogs(t *testing.T) {
	docker := &fakeDockerAPIClient{
		inspectFunc: func(string) (dockerclient.ContainerInspectResult, error) {
			return dockerclient.ContainerInspectResult{}, errdefs.ErrNotFound.WithMessage("No such container: abc123")
		},
	}
	cli := test.NewFakeCli(docker)

	err := streamRunLogs(context.Background(), cli, "abc123", false)

	assert.ErrorContains(t, err, "logs are no longer available")
}
