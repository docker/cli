package service

import (
	"testing"

	"github.com/moby/moby/api/types/swarm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestPickTask(t *testing.T) {
	running := swarm.TaskStatus{
		State:           swarm.TaskStateRunning,
		ContainerStatus: &swarm.ContainerStatus{ContainerID: "c2"},
	}
	tasks := []swarm.Task{
		{ID: "t1", NodeID: "n1", Status: swarm.TaskStatus{State: swarm.TaskStateFailed}},
		{ID: "t2", NodeID: "n2", Status: running},
		{ID: "t3", NodeID: "n3", Status: running},
	}

	t.Run("first running task by default", func(t *testing.T) {
		task, err := pickTask(tasks, "")
		assert.NilError(t, err)
		assert.Check(t, is.Equal(task.ID, "t2"))
	})

	t.Run("explicit task id", func(t *testing.T) {
		task, err := pickTask(tasks, "t3")
		assert.NilError(t, err)
		assert.Check(t, is.Equal(task.ID, "t3"))
	})

	t.Run("explicit task id not running", func(t *testing.T) {
		_, err := pickTask(tasks, "t1")
		assert.ErrorContains(t, err, "not running")
	})

	t.Run("explicit task id not found", func(t *testing.T) {
		_, err := pickTask(tasks, "nope")
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("no running task", func(t *testing.T) {
		_, err := pickTask(nil, "")
		assert.ErrorContains(t, err, "no running task")
	})

	t.Run("running task without container status", func(t *testing.T) {
		_, err := pickTask([]swarm.Task{
			{ID: "t4", Status: swarm.TaskStatus{State: swarm.TaskStateRunning}},
		}, "")
		assert.ErrorContains(t, err, "no running task")
	})
}

func TestNodeSSHHost(t *testing.T) {
	t.Run("prefers status addr", func(t *testing.T) {
		n := swarm.Node{Status: swarm.NodeStatus{Addr: "10.0.0.5"}}
		assert.Check(t, is.Equal(nodeSSHHost(n, ""), "ssh://10.0.0.5"))
	})

	t.Run("falls back to hostname on zero addr", func(t *testing.T) {
		n := swarm.Node{
			Status:      swarm.NodeStatus{Addr: "0.0.0.0"},
			Description: swarm.NodeDescription{Hostname: "pallas"},
		}
		assert.Check(t, is.Equal(nodeSSHHost(n, ""), "ssh://pallas"))
	})

	t.Run("ssh user is prepended", func(t *testing.T) {
		n := swarm.Node{Status: swarm.NodeStatus{Addr: "10.0.0.5"}}
		assert.Check(t, is.Equal(nodeSSHHost(n, "core"), "ssh://core@10.0.0.5"))
	})
}
