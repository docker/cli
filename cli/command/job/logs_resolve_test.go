package job

import (
	"errors"
	"testing"

	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestResolveLogsRunNeverRun(t *testing.T) {
	job := &jobsv0.Job{Name: "migrate", LatestRun: nil}
	fallbackCalled := false
	listRuns := func() (*jobsv0.ListRunsReply, error) {
		fallbackCalled = true
		return &jobsv0.ListRunsReply{}, nil
	}

	run, err := resolveLogsRun(job, listRuns)

	assert.Check(t, is.Nil(run))
	assert.ErrorContains(t, err, `job "migrate" has never run`)
	assert.Check(t, !fallbackCalled, "listRuns must not be called when the job never ran")
}

func TestResolveLogsRunReturnsLatestRunWhenContainerPresent(t *testing.T) {
	latest := &jobsv0.Run{ID: "run-3", ContainerID: "abc123", ContainerGone: false}
	job := &jobsv0.Job{Name: "backup", LatestRun: latest}
	fallbackCalled := false
	listRuns := func() (*jobsv0.ListRunsReply, error) {
		fallbackCalled = true
		return &jobsv0.ListRunsReply{}, nil
	}

	run, err := resolveLogsRun(job, listRuns)

	assert.NilError(t, err)
	assert.Check(t, is.Equal(run.ID, "run-3"))
	assert.Check(t, is.Equal(run.ContainerID, "abc123"))
	assert.Check(t, !fallbackCalled, "listRuns must not be called when LatestRun's container is not gone")
}

func TestResolveLogsRunFallsBackToListRunsWhenContainerGone(t *testing.T) {
	latest := &jobsv0.Run{ID: "run-3", ContainerGone: true}
	job := &jobsv0.Job{Name: "backup", LatestRun: latest}
	listRuns := func() (*jobsv0.ListRunsReply, error) {
		// Most recent first, matching ListRuns's documented ordering:
		// run-3 (the gone latest run) is included in history too, run-2's
		// container was also removed, run-1's container is still around.
		return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{
			{ID: "run-3", ContainerGone: true},
			{ID: "run-2", ContainerGone: true},
			{ID: "run-1", ContainerID: "def456", ContainerGone: false},
		}}, nil
	}

	run, err := resolveLogsRun(job, listRuns)

	assert.NilError(t, err)
	assert.Check(t, is.Equal(run.ID, "run-1"))
	assert.Check(t, is.Equal(run.ContainerID, "def456"))
}

func TestResolveLogsRunNoLogsAvailableWhenNoRunHasContainer(t *testing.T) {
	latest := &jobsv0.Run{ID: "run-2", ContainerGone: true}
	job := &jobsv0.Job{Name: "backup", LatestRun: latest}
	listRuns := func() (*jobsv0.ListRunsReply, error) {
		return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{
			{ID: "run-2", ContainerGone: true},
			{ID: "run-1", ContainerGone: true},
		}}, nil
	}

	run, err := resolveLogsRun(job, listRuns)

	assert.Check(t, is.Nil(run))
	assert.ErrorContains(t, err, `no logs available for job "backup"`)
}

func TestResolveLogsRunNoLogsAvailableWhenHistoryEmpty(t *testing.T) {
	latest := &jobsv0.Run{ID: "run-1", ContainerGone: true}
	job := &jobsv0.Job{Name: "backup", LatestRun: latest}
	listRuns := func() (*jobsv0.ListRunsReply, error) {
		return &jobsv0.ListRunsReply{}, nil
	}

	run, err := resolveLogsRun(job, listRuns)

	assert.Check(t, is.Nil(run))
	assert.ErrorContains(t, err, `no logs available for job "backup"`)
}

func TestResolveLogsRunPropagatesListRunsError(t *testing.T) {
	latest := &jobsv0.Run{ID: "run-1", ContainerGone: true}
	job := &jobsv0.Job{Name: "backup", LatestRun: latest}
	listRuns := func() (*jobsv0.ListRunsReply, error) {
		return nil, errors.New("connection refused")
	}

	run, err := resolveLogsRun(job, listRuns)

	assert.Check(t, is.Nil(run))
	assert.ErrorContains(t, err, "connection refused")
}

func TestResolveLogsRunFallsBackWhenLatestRunHasNoContainerYet(t *testing.T) {
	// A pending run, or one that failed before its container was created, has
	// neither a ContainerID nor ContainerGone set.
	job := &jobsv0.Job{Name: "backup", LatestRun: &jobsv0.Run{ID: "run-2", State: jobsv0.RunStateFailed, Error: "no such image"}}
	listRuns := func() (*jobsv0.ListRunsReply, error) {
		return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{
			{ID: "run-2", State: jobsv0.RunStateFailed},
			{ID: "run-1", ContainerID: "def456"},
		}}, nil
	}

	run, err := resolveLogsRun(job, listRuns)

	assert.NilError(t, err)
	assert.Check(t, is.Equal(run.ID, "run-1"))
}

func TestResolveLogsRunNoLogsAvailableWhenNoRunEverHadContainer(t *testing.T) {
	job := &jobsv0.Job{Name: "backup", LatestRun: &jobsv0.Run{ID: "run-1", State: jobsv0.RunStatePending}}
	listRuns := func() (*jobsv0.ListRunsReply, error) {
		return &jobsv0.ListRunsReply{Runs: []jobsv0.Run{{ID: "run-1", State: jobsv0.RunStatePending}}}, nil
	}

	run, err := resolveLogsRun(job, listRuns)

	assert.Check(t, is.Nil(run))
	assert.ErrorContains(t, err, `no logs available for job "backup"`)
}
