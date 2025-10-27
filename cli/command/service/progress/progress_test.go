package progress

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client/pkg/progress"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type mockProgress struct {
	p []progress.Progress
}

func (mp *mockProgress) WriteProgress(p progress.Progress) error {
	mp.p = append(mp.p, p)
	return nil
}

func (mp *mockProgress) clear() {
	mp.p = nil
}

type updaterTester struct {
	t           *testing.T
	updater     progressUpdater
	p           *mockProgress
	service     swarm.Service
	activeNodes map[string]struct{}
	rollback    bool
}

func (u updaterTester) testUpdater(tasks []swarm.Task, expectedConvergence bool, expectedProgress []progress.Progress) {
	u.p.clear()

	converged, err := u.updater.update(u.service, tasks, u.activeNodes, u.rollback)
	assert.Check(u.t, err)
	assert.Check(u.t, is.Equal(expectedConvergence, converged))
	assert.Check(u.t, is.DeepEqual(expectedProgress, u.p.p))
}

func (u updaterTester) testUpdaterNoOrder(tasks []swarm.Task, expectedConvergence bool, expectedProgress []progress.Progress) {
	u.p.clear()
	converged, err := u.updater.update(u.service, tasks, u.activeNodes, u.rollback)
	assert.Check(u.t, err)
	assert.Check(u.t, is.Equal(expectedConvergence, converged))

	// instead of checking that expected and actual match exactly, verify that
	// they are the same length, and every time from actual is in expected.
	assert.Check(u.t, is.Equal(len(expectedProgress), len(u.p.p)))
	for _, prog := range expectedProgress {
		assert.Check(u.t, is.Contains(u.p.p, prog))
	}
}

func TestReplicatedProgressUpdaterOneReplica(t *testing.T) {
	replicas := uint64(1)

	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				Replicated: &swarm.ReplicatedService{
					Replicas: &replicas,
				},
			},
		},
	}

	p := &mockProgress{}
	ut := updaterTester{
		t: t,
		updater: &replicatedProgressUpdater{
			progressOut: p,
		},
		p:           p,
		activeNodes: map[string]struct{}{"a": {}, "b": {}},
		service:     service,
	}

	tasks := []swarm.Task{}

	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: "0 out of 1 tasks"},
			{ID: "1/1", Action: " "},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// Task with DesiredState beyond Running is ignored
	tasks = append(tasks,
		swarm.Task{
			ID:           "1",
			NodeID:       "a",
			DesiredState: swarm.TaskStateShutdown,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
		})
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// Task with valid DesiredState and State updates progress bar
	tasks[0].DesiredState = swarm.TaskStateRunning
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "1/1", Action: "new      ", Current: 1, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// If the task exposes an error, we should show that instead of the
	// progress bar.
	tasks[0].Status.Err = "something is wrong"
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "1/1", Action: "something is wrong"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// When the task reaches running, update should return true
	tasks[0].Status.Err = ""
	tasks[0].Status.State = swarm.TaskStateRunning
	ut.testUpdater(tasks, true,
		[]progress.Progress{
			{ID: "1/1", Action: "running  ", Current: 9, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "1 out of 1 tasks"},
		})

	// If the task fails, update should return false again
	tasks[0].Status.Err = "task failed"
	tasks[0].Status.State = swarm.TaskStateFailed
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "1/1", Action: "task failed"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// If the task is restarted, progress output should be shown for the
	// replacement task, not the old task.
	tasks[0].DesiredState = swarm.TaskStateShutdown
	tasks = append(tasks,
		swarm.Task{
			ID:           "2",
			NodeID:       "b",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		})
	ut.testUpdater(tasks, true,
		[]progress.Progress{
			{ID: "1/1", Action: "running  ", Current: 9, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "1 out of 1 tasks"},
		})

	// Add a new task while the current one is still running, to simulate
	// "start-then-stop" updates.
	tasks = append(tasks,
		swarm.Task{
			ID:           "3",
			NodeID:       "b",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStatePreparing},
		})
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "1/1", Action: "preparing", Current: 6, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})
}

func TestReplicatedProgressUpdaterManyReplicas(t *testing.T) {
	replicas := uint64(50)

	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				Replicated: &swarm.ReplicatedService{
					Replicas: &replicas,
				},
			},
		},
	}

	p := &mockProgress{}
	ut := updaterTester{
		t: t,
		updater: &replicatedProgressUpdater{
			progressOut: p,
		},
		p:           p,
		activeNodes: map[string]struct{}{"a": {}, "b": {}},
		service:     service,
	}

	tasks := []swarm.Task{}

	// No per-task progress bars because there are too many replicas
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: fmt.Sprintf("0 out of %d tasks", replicas)},
			{ID: "overall progress", Action: fmt.Sprintf("0 out of %d tasks", replicas)},
		})

	for i := 0; i != int(replicas); i++ {
		tasks = append(tasks,
			swarm.Task{
				ID:           strconv.Itoa(i),
				Slot:         i + 1,
				NodeID:       "a",
				DesiredState: swarm.TaskStateRunning,
				Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			})

		if i%2 == 1 {
			tasks[i].NodeID = "b"
		}
		ut.testUpdater(tasks, false,
			[]progress.Progress{
				{ID: "overall progress", Action: fmt.Sprintf("%d out of %d tasks", i, replicas)},
			})

		tasks[i].Status.State = swarm.TaskStateRunning
		ut.testUpdater(tasks, uint64(i) == replicas-1,
			[]progress.Progress{
				{ID: "overall progress", Action: fmt.Sprintf("%d out of %d tasks", i+1, replicas)},
			})
	}
}

func TestGlobalProgressUpdaterOneNode(t *testing.T) {
	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				Global: &swarm.GlobalService{},
			},
		},
	}

	p := &mockProgress{}
	ut := updaterTester{
		t: t,
		updater: &globalProgressUpdater{
			progressOut: p,
		},
		p:           p,
		activeNodes: map[string]struct{}{"a": {}, "b": {}},
		service:     service,
	}

	tasks := []swarm.Task{}

	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: "waiting for new tasks"},
		})

	// Task with DesiredState beyond Running is ignored
	tasks = append(tasks,
		swarm.Task{
			ID:           "1",
			NodeID:       "a",
			DesiredState: swarm.TaskStateShutdown,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
		})
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: "0 out of 1 tasks"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// Task with valid DesiredState and State updates progress bar
	tasks[0].DesiredState = swarm.TaskStateRunning
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "a", Action: "new      ", Current: 1, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// If the task exposes an error, we should show that instead of the
	// progress bar.
	tasks[0].Status.Err = "something is wrong"
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "a", Action: "something is wrong"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// When the task reaches running, update should return true
	tasks[0].Status.Err = ""
	tasks[0].Status.State = swarm.TaskStateRunning
	ut.testUpdater(tasks, true,
		[]progress.Progress{
			{ID: "a", Action: "running  ", Current: 9, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "1 out of 1 tasks"},
		})

	// If the task fails, update should return false again
	tasks[0].Status.Err = "task failed"
	tasks[0].Status.State = swarm.TaskStateFailed
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "a", Action: "task failed"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// If the task is restarted, progress output should be shown for the
	// replacement task, not the old task.
	tasks[0].DesiredState = swarm.TaskStateShutdown
	tasks = append(tasks,
		swarm.Task{
			ID:           "2",
			NodeID:       "a",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		})
	ut.testUpdater(tasks, true,
		[]progress.Progress{
			{ID: "a", Action: "running  ", Current: 9, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "1 out of 1 tasks"},
		})

	// Add a new task while the current one is still running, to simulate
	// "start-then-stop" updates.
	tasks = append(tasks,
		swarm.Task{
			ID:           "3",
			NodeID:       "a",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStatePreparing},
		})
	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "a", Action: "preparing", Current: 6, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})
}

func TestGlobalProgressUpdaterManyNodes(t *testing.T) {
	nodes := 50

	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				Global: &swarm.GlobalService{},
			},
		},
	}

	p := &mockProgress{}
	ut := updaterTester{
		t: t,
		updater: &globalProgressUpdater{
			progressOut: p,
		},
		p:           p,
		activeNodes: map[string]struct{}{},
		service:     service,
	}

	for i := 0; i != nodes; i++ {
		ut.activeNodes[strconv.Itoa(i)] = struct{}{}
	}

	tasks := []swarm.Task{}

	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: "waiting for new tasks"},
		})

	for i := 0; i != nodes; i++ {
		tasks = append(tasks,
			swarm.Task{
				ID:           "task" + strconv.Itoa(i),
				NodeID:       strconv.Itoa(i),
				DesiredState: swarm.TaskStateRunning,
				Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			})
	}

	ut.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: fmt.Sprintf("0 out of %d tasks", nodes)},
			{ID: "overall progress", Action: fmt.Sprintf("0 out of %d tasks", nodes)},
		})

	for i := 0; i != nodes; i++ {
		tasks[i].Status.State = swarm.TaskStateRunning
		ut.testUpdater(tasks, i == nodes-1,
			[]progress.Progress{
				{ID: "overall progress", Action: fmt.Sprintf("%d out of %d tasks", i+1, nodes)},
			})
	}
}

func TestReplicatedJobProgressUpdaterSmall(t *testing.T) {
	concurrent := uint64(2)
	total := uint64(5)

	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				ReplicatedJob: &swarm.ReplicatedJob{
					MaxConcurrent:    &concurrent,
					TotalCompletions: &total,
				},
			},
		},
		JobStatus: &swarm.JobStatus{
			JobIteration: swarm.Version{Index: 1},
		},
	}

	p := &mockProgress{}
	ut := updaterTester{
		t:           t,
		updater:     newReplicatedJobProgressUpdater(service, p),
		p:           p,
		activeNodes: map[string]struct{}{"a": {}, "b": {}},
		service:     service,
	}

	// create some tasks belonging to a previous iteration
	tasks := []swarm.Task{
		{
			ID:           "oldtask1",
			Slot:         0,
			NodeID:       "",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: 0},
		}, {
			ID:           "oldtask2",
			Slot:         1,
			NodeID:       "",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateComplete},
			JobIteration: &swarm.Version{Index: 0},
		},
	}

	ut.testUpdater(tasks, false, []progress.Progress{
		// on the initial pass, we draw all of the progress bars at once, which
		// puts them in order for the rest of the operation
		{ID: "job progress", Action: "0 out of 5 complete", Current: 0, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "0 out of 2 tasks"},
		{ID: "1/5", Action: " "},
		{ID: "2/5", Action: " "},
		{ID: "3/5", Action: " "},
		{ID: "4/5", Action: " "},
		{ID: "5/5", Action: " "},
		// from here on, we draw as normal. as a side effect, we will have a
		// second update for the job progress and active tasks. This has no
		// practical effect on the UI, it's just a side effect of the update
		// logic.
		{ID: "job progress", Action: "0 out of 5 complete", Current: 0, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "0 out of 2 tasks"},
	})

	// wipe the old tasks out of the list
	tasks = []swarm.Task{}
	tasks = append(tasks,
		swarm.Task{
			ID:           "task1",
			Slot:         0,
			NodeID:       "",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: service.JobStatus.JobIteration.Index},
		},
		swarm.Task{
			ID:           "task2",
			Slot:         1,
			NodeID:       "",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: service.JobStatus.JobIteration.Index},
		},
	)
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "1/5", Action: "new      ", Current: 1, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "new      ", Current: 1, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "0 out of 5 complete", Current: 0, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "2 out of 2 tasks"},
	})

	tasks[0].Status.State = swarm.TaskStatePreparing
	tasks[1].Status.State = swarm.TaskStateAssigned
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "1/5", Action: "preparing", Current: 6, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "assigned ", Current: 4, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "0 out of 5 complete", Current: 0, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "2 out of 2 tasks"},
	})

	tasks[0].Status.State = swarm.TaskStateRunning
	tasks[1].Status.State = swarm.TaskStatePreparing
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "1/5", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "preparing", Current: 6, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "0 out of 5 complete", Current: 0, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "2 out of 2 tasks"},
	})

	tasks[0].Status.State = swarm.TaskStateComplete
	tasks[1].Status.State = swarm.TaskStateComplete
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "1/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "2 out of 5 complete", Current: 2, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "0 out of 2 tasks"},
	})

	tasks = append(tasks,
		swarm.Task{
			ID:           "task3",
			Slot:         2,
			NodeID:       "",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: service.JobStatus.JobIteration.Index},
		},
		swarm.Task{
			ID:           "task4",
			Slot:         3,
			NodeID:       "",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: service.JobStatus.JobIteration.Index},
		},
	)

	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "1/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "3/5", Action: "new      ", Current: 1, Total: 10, HideCounts: true},
		{ID: "4/5", Action: "new      ", Current: 1, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "2 out of 5 complete", Current: 2, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "2 out of 2 tasks"},
	})

	tasks[2].Status.State = swarm.TaskStateRunning
	tasks[3].Status.State = swarm.TaskStateRunning
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "1/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "3/5", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "4/5", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "2 out of 5 complete", Current: 2, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "2 out of 2 tasks"},
	})

	tasks[3].Status.State = swarm.TaskStateComplete
	tasks = append(tasks,
		swarm.Task{
			ID:           "task5",
			Slot:         4,
			NodeID:       "",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
			JobIteration: &swarm.Version{Index: service.JobStatus.JobIteration.Index},
		},
	)
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "1/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "3/5", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "4/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "5/5", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "3 out of 5 complete", Current: 3, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "2 out of 2 tasks"},
	})

	tasks[2].Status.State = swarm.TaskStateFailed
	tasks[2].Status.Err = "the task failed"
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "1/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "3/5", Action: "the task failed"},
		{ID: "4/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "5/5", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "3 out of 5 complete", Current: 3, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "1 out of 2 tasks"},
	})

	tasks[4].Status.State = swarm.TaskStateComplete
	tasks = append(tasks,
		swarm.Task{
			ID:           "task6",
			Slot:         2,
			NodeID:       "",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
			JobIteration: &swarm.Version{Index: service.JobStatus.JobIteration.Index},
		},
	)
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "1/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "3/5", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "4/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "5/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "4 out of 5 complete", Current: 4, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "1 out of 1 tasks"},
	})

	tasks[5].Status.State = swarm.TaskStateComplete
	ut.testUpdater(tasks, true, []progress.Progress{
		{ID: "1/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "2/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "3/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "4/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "5/5", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "5 out of 5 complete", Current: 5, Total: 5, HideCounts: true},
		{ID: "active tasks", Action: "0 out of 0 tasks"},
	})
}

func TestReplicatedJobProgressUpdaterLarge(t *testing.T) {
	concurrent := uint64(10)
	total := uint64(50)

	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				ReplicatedJob: &swarm.ReplicatedJob{
					MaxConcurrent:    &concurrent,
					TotalCompletions: &total,
				},
			},
		},
		JobStatus: &swarm.JobStatus{
			JobIteration: swarm.Version{Index: 0},
		},
	}

	p := &mockProgress{}
	ut := updaterTester{
		t:           t,
		updater:     newReplicatedJobProgressUpdater(service, p),
		p:           p,
		activeNodes: map[string]struct{}{"a": {}, "b": {}},
		service:     service,
	}

	tasks := []swarm.Task{}

	// see the comments in TestReplicatedJobProgressUpdaterSmall for why
	// we write this out twice.
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "job progress", Action: " 0 out of 50 complete", Current: 0, Total: 50, HideCounts: true},
		{ID: "active tasks", Action: " 0 out of 10 tasks"},
		// we don't write out individual status bars for a large job, only the
		// overall progress bar
		{ID: "job progress", Action: " 0 out of 50 complete", Current: 0, Total: 50, HideCounts: true},
		{ID: "active tasks", Action: " 0 out of 10 tasks"},
	})

	// first, create the initial batch of running tasks
	for i := 0; i < int(concurrent); i++ {
		tasks = append(tasks, swarm.Task{
			ID:           strconv.Itoa(i),
			Slot:         i,
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: 0},
		})

		ut.testUpdater(tasks, false, []progress.Progress{
			{ID: "job progress", Action: " 0 out of 50 complete", Current: 0, Total: 50, HideCounts: true},
			{ID: "active tasks", Action: fmt.Sprintf("%2d out of 10 tasks", i+1)},
		})
	}

	// now, start moving tasks to completed, and starting new tasks after them.
	// to do this, we'll start at 0, mark a task complete, and then append a
	// new one. we'll stop before we get to the end, because the end has a
	// steadily decreasing denominator for the active tasks
	//
	// for 10 concurrent 50 total, this means we'll stop at 50 - 10 = 40 tasks
	// in the completed state, 10 tasks running. the last index in use will be
	// 39.
	for i := 0; i < int(total)-int(concurrent); i++ {
		tasks[i].Status.State = swarm.TaskStateComplete
		ut.testUpdater(tasks, false, []progress.Progress{
			{ID: "job progress", Action: fmt.Sprintf("%2d out of 50 complete", i+1), Current: int64(i + 1), Total: 50, HideCounts: true},
			{ID: "active tasks", Action: " 9 out of 10 tasks"},
		})

		last := len(tasks)
		tasks = append(tasks, swarm.Task{
			ID:           strconv.Itoa(last),
			Slot:         last,
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: 0},
		})

		ut.testUpdater(tasks, false, []progress.Progress{
			{ID: "job progress", Action: fmt.Sprintf("%2d out of 50 complete", i+1), Current: int64(i + 1), Total: 50, HideCounts: true},
			{ID: "active tasks", Action: "10 out of 10 tasks"},
		})
	}

	// quick check, to make sure we did the math right when we wrote this code:
	// we do have 50 tasks in the slice, right?
	assert.Check(t, is.Equal(len(tasks), int(total)))

	// now, we're down to our last 10 tasks, which are all running. We need to
	// wind these down
	for i := int(total) - int(concurrent) - 1; i < int(total); i++ {
		tasks[i].Status.State = swarm.TaskStateComplete
		ut.testUpdater(tasks, (i+1 == int(total)), []progress.Progress{
			{ID: "job progress", Action: fmt.Sprintf("%2d out of 50 complete", i+1), Current: int64(i + 1), Total: 50, HideCounts: true},
			{ID: "active tasks", Action: fmt.Sprintf("%2[1]d out of %2[1]d tasks", int(total)-(i+1))},
		})
	}
}

func TestGlobalJobProgressUpdaterSmall(t *testing.T) {
	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				GlobalJob: &swarm.GlobalJob{},
			},
		},
		JobStatus: &swarm.JobStatus{
			JobIteration: swarm.Version{Index: 1},
		},
	}

	p := &mockProgress{}
	ut := updaterTester{
		t: t,
		updater: &globalJobProgressUpdater{
			progressOut: p,
		},
		p:           p,
		activeNodes: map[string]struct{}{"a": {}, "b": {}, "c": {}},
		service:     service,
	}

	tasks := []swarm.Task{
		{
			ID:           "oldtask1",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateComplete},
			JobIteration: &swarm.Version{Index: 0},
			NodeID:       "a",
		}, {
			ID:           "oldtask2",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateComplete},
			JobIteration: &swarm.Version{Index: 0},
			NodeID:       "b",
		}, {
			ID:           "task1",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: 1},
			NodeID:       "a",
		}, {
			ID:           "task2",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: 1},
			NodeID:       "b",
		}, {
			ID:           "task3",
			DesiredState: swarm.TaskStateComplete,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			JobIteration: &swarm.Version{Index: 1},
			NodeID:       "c",
		},
	}

	// we don't know how many tasks will be created until we get the initial
	// task list, so we should not write out any definitive answers yet.
	ut.testUpdater([]swarm.Task{}, false, []progress.Progress{
		{ID: "job progress", Action: "waiting for tasks"},
	})

	ut.testUpdaterNoOrder(tasks, false, []progress.Progress{
		{ID: "job progress", Action: "0 out of 3 complete", Current: 0, Total: 3, HideCounts: true},
		{ID: "a", Action: "new      ", Current: 1, Total: 10, HideCounts: true},
		{ID: "b", Action: "new      ", Current: 1, Total: 10, HideCounts: true},
		{ID: "c", Action: "new      ", Current: 1, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "0 out of 3 complete", Current: 0, Total: 3, HideCounts: true},
	})

	tasks[2].Status.State = swarm.TaskStatePreparing
	tasks[3].Status.State = swarm.TaskStateRunning
	tasks[4].Status.State = swarm.TaskStateAccepted
	ut.testUpdaterNoOrder(tasks, false, []progress.Progress{
		{ID: "a", Action: "preparing", Current: 6, Total: 10, HideCounts: true},
		{ID: "b", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "c", Action: "accepted ", Current: 5, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "0 out of 3 complete", Current: 0, Total: 3, HideCounts: true},
	})

	tasks[2].Status.State = swarm.TaskStateRunning
	tasks[3].Status.State = swarm.TaskStateComplete
	tasks[4].Status.State = swarm.TaskStateRunning
	ut.testUpdaterNoOrder(tasks, false, []progress.Progress{
		{ID: "a", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "b", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "c", Action: "running  ", Current: 9, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "1 out of 3 complete", Current: 1, Total: 3, HideCounts: true},
	})

	tasks[2].Status.State = swarm.TaskStateFailed
	tasks[2].Status.Err = "task failed"
	tasks[4].Status.State = swarm.TaskStateComplete
	ut.testUpdaterNoOrder(tasks, false, []progress.Progress{
		{ID: "a", Action: "task failed"},
		{ID: "b", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "c", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "2 out of 3 complete", Current: 2, Total: 3, HideCounts: true},
	})

	tasks = append(tasks, swarm.Task{
		ID:           "task4",
		DesiredState: swarm.TaskStateComplete,
		Status:       swarm.TaskStatus{State: swarm.TaskStatePreparing},
		NodeID:       tasks[2].NodeID,
		JobIteration: &swarm.Version{Index: 1},
	})

	ut.testUpdaterNoOrder(tasks, false, []progress.Progress{
		{ID: "a", Action: "preparing", Current: 6, Total: 10, HideCounts: true},
		{ID: "b", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "c", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "2 out of 3 complete", Current: 2, Total: 3, HideCounts: true},
	})

	tasks[5].Status.State = swarm.TaskStateComplete
	ut.testUpdaterNoOrder(tasks, true, []progress.Progress{
		{ID: "a", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "b", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "c", Action: "complete ", Current: 10, Total: 10, HideCounts: true},
		{ID: "job progress", Action: "3 out of 3 complete", Current: 3, Total: 3, HideCounts: true},
	})
}

func TestGlobalJobProgressUpdaterLarge(t *testing.T) {
	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				GlobalJob: &swarm.GlobalJob{},
			},
		},
		JobStatus: &swarm.JobStatus{
			JobIteration: swarm.Version{Index: 1},
		},
	}

	activeNodes := map[string]struct{}{}
	for i := 0; i < 50; i++ {
		activeNodes[fmt.Sprintf("node%v", i)] = struct{}{}
	}

	p := &mockProgress{}
	ut := updaterTester{
		t: t,
		updater: &globalJobProgressUpdater{
			progressOut: p,
		},
		p:           p,
		activeNodes: activeNodes,
		service:     service,
	}

	tasks := []swarm.Task{}
	for nodeID := range activeNodes {
		tasks = append(tasks, swarm.Task{
			ID:           "task" + nodeID,
			NodeID:       nodeID,
			DesiredState: swarm.TaskStateComplete,
			Status: swarm.TaskStatus{
				State: swarm.TaskStateNew,
			},
			JobIteration: &swarm.Version{Index: 1},
		})
	}

	// no bars, because too many tasks
	ut.testUpdater(tasks, false, []progress.Progress{
		{ID: "job progress", Action: " 0 out of 50 complete", Current: 0, Total: 50, HideCounts: true},
		{ID: "job progress", Action: " 0 out of 50 complete", Current: 0, Total: 50, HideCounts: true},
	})

	for i := range tasks {
		tasks[i].Status.State = swarm.TaskStateComplete
		ut.testUpdater(tasks, i+1 == len(activeNodes), []progress.Progress{
			{
				ID:      "job progress",
				Action:  fmt.Sprintf("%2d out of 50 complete", i+1),
				Current: int64(i + 1), Total: 50, HideCounts: true,
			},
		})
	}
}
