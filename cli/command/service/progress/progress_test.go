package progress

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/progress"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
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
	updaterTester := updaterTester{
		t: t,
		updater: &replicatedProgressUpdater{
			progressOut: p,
		},
		p:           p,
		activeNodes: map[string]struct{}{"a": {}, "b": {}},
		service:     service,
	}

	tasks := []swarm.Task{}

	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: "0 out of 1 tasks"},
			{ID: "1/1", Action: " "},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// Task with DesiredState beyond Running is ignored
	tasks = append(tasks,
		swarm.Task{ID: "1",
			NodeID:       "a",
			DesiredState: swarm.TaskStateShutdown,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
		})
	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// Task with valid DesiredState and State updates progress bar
	tasks[0].DesiredState = swarm.TaskStateRunning
	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "1/1", Action: "new      ", Current: 1, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// If the task exposes an error, we should show that instead of the
	// progress bar.
	tasks[0].Status.Err = "something is wrong"
	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "1/1", Action: "something is wrong"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// When the task reaches running, update should return true
	tasks[0].Status.Err = ""
	tasks[0].Status.State = swarm.TaskStateRunning
	updaterTester.testUpdater(tasks, true,
		[]progress.Progress{
			{ID: "1/1", Action: "running  ", Current: 9, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "1 out of 1 tasks"},
		})

	// If the task fails, update should return false again
	tasks[0].Status.Err = "task failed"
	tasks[0].Status.State = swarm.TaskStateFailed
	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "1/1", Action: "task failed"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// If the task is restarted, progress output should be shown for the
	// replacement task, not the old task.
	tasks[0].DesiredState = swarm.TaskStateShutdown
	tasks = append(tasks,
		swarm.Task{ID: "2",
			NodeID:       "b",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		})
	updaterTester.testUpdater(tasks, true,
		[]progress.Progress{
			{ID: "1/1", Action: "running  ", Current: 9, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "1 out of 1 tasks"},
		})

	// Add a new task while the current one is still running, to simulate
	// "start-then-stop" updates.
	tasks = append(tasks,
		swarm.Task{ID: "3",
			NodeID:       "b",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStatePreparing},
		})
	updaterTester.testUpdater(tasks, false,
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
	updaterTester := updaterTester{
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
	updaterTester.testUpdater(tasks, false,
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
		updaterTester.testUpdater(tasks, false,
			[]progress.Progress{
				{ID: "overall progress", Action: fmt.Sprintf("%d out of %d tasks", i, replicas)},
			})

		tasks[i].Status.State = swarm.TaskStateRunning
		updaterTester.testUpdater(tasks, uint64(i) == replicas-1,
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
	updaterTester := updaterTester{
		t: t,
		updater: &globalProgressUpdater{
			progressOut: p,
		},
		p:           p,
		activeNodes: map[string]struct{}{"a": {}, "b": {}},
		service:     service,
	}

	tasks := []swarm.Task{}

	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: "waiting for new tasks"},
		})

	// Task with DesiredState beyond Running is ignored
	tasks = append(tasks,
		swarm.Task{ID: "1",
			NodeID:       "a",
			DesiredState: swarm.TaskStateShutdown,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
		})
	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: "0 out of 1 tasks"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// Task with valid DesiredState and State updates progress bar
	tasks[0].DesiredState = swarm.TaskStateRunning
	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "a", Action: "new      ", Current: 1, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// If the task exposes an error, we should show that instead of the
	// progress bar.
	tasks[0].Status.Err = "something is wrong"
	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "a", Action: "something is wrong"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// When the task reaches running, update should return true
	tasks[0].Status.Err = ""
	tasks[0].Status.State = swarm.TaskStateRunning
	updaterTester.testUpdater(tasks, true,
		[]progress.Progress{
			{ID: "a", Action: "running  ", Current: 9, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "1 out of 1 tasks"},
		})

	// If the task fails, update should return false again
	tasks[0].Status.Err = "task failed"
	tasks[0].Status.State = swarm.TaskStateFailed
	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "a", Action: "task failed"},
			{ID: "overall progress", Action: "0 out of 1 tasks"},
		})

	// If the task is restarted, progress output should be shown for the
	// replacement task, not the old task.
	tasks[0].DesiredState = swarm.TaskStateShutdown
	tasks = append(tasks,
		swarm.Task{ID: "2",
			NodeID:       "a",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		})
	updaterTester.testUpdater(tasks, true,
		[]progress.Progress{
			{ID: "a", Action: "running  ", Current: 9, Total: 9, HideCounts: true},
			{ID: "overall progress", Action: "1 out of 1 tasks"},
		})

	// Add a new task while the current one is still running, to simulate
	// "start-then-stop" updates.
	tasks = append(tasks,
		swarm.Task{ID: "3",
			NodeID:       "a",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStatePreparing},
		})
	updaterTester.testUpdater(tasks, false,
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
	updaterTester := updaterTester{
		t: t,
		updater: &globalProgressUpdater{
			progressOut: p,
		},
		p:           p,
		activeNodes: map[string]struct{}{},
		service:     service,
	}

	for i := 0; i != nodes; i++ {
		updaterTester.activeNodes[strconv.Itoa(i)] = struct{}{}
	}

	tasks := []swarm.Task{}

	updaterTester.testUpdater(tasks, false,
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

	updaterTester.testUpdater(tasks, false,
		[]progress.Progress{
			{ID: "overall progress", Action: fmt.Sprintf("0 out of %d tasks", nodes)},
			{ID: "overall progress", Action: fmt.Sprintf("0 out of %d tasks", nodes)},
		})

	for i := 0; i != nodes; i++ {
		tasks[i].Status.State = swarm.TaskStateRunning
		updaterTester.testUpdater(tasks, i == nodes-1,
			[]progress.Progress{
				{ID: "overall progress", Action: fmt.Sprintf("%d out of %d tasks", i+1, nodes)},
			})
	}
}
