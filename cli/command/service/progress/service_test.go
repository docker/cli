package progress

import (
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

type fakeClient struct {
	client.Client
	serviceInspectFunc func(string, types.ServiceInspectOptions) (swarm.Service, []byte, error)
	taskListFunc       func(types.TaskListOptions) ([]swarm.Task, error)
	nodeListFunc       func(types.NodeListOptions) ([]swarm.Node, error)
}

func (f *fakeClient) ServiceInspectWithRaw(_ context.Context, serviceID string, options types.ServiceInspectOptions) (swarm.Service, []byte, error) {
	if f.serviceInspectFunc != nil {
		return f.serviceInspectFunc(serviceID, options)
	}
	return swarm.Service{}, nil, nil
}

func (f *fakeClient) TaskList(_ context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	if f.taskListFunc != nil {
		return f.taskListFunc(options)
	}
	return nil, nil
}

func (f *fakeClient) NodeList(_ context.Context, options types.NodeListOptions) ([]swarm.Node, error) {
	if f.nodeListFunc != nil {
		return f.nodeListFunc(options)
	}
	return nil, nil
}

func TestWaitOnServiceWithTimeout(t *testing.T) {
	serviceID := "abcdef"
	timeout := 10 * time.Millisecond
	client := &fakeClient{
		serviceInspectFunc: func(string, types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return replicastedService(), nil, nil
		},
	}

	opts := WaitOnServiceOptions{Timeout: &timeout}
	err := WaitOnService(context.Background(), test.NewFakeCli(client), serviceID, opts)
	assert.EqualError(t, err, "timeout (10ms) waiting on abcdef to converge. "+msgOperationContinuingInBackground)
}

func replicastedService() swarm.Service {
	return swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				Replicated: &swarm.ReplicatedService{Replicas: unit64Ptr(5)},
			},
			UpdateConfig: &swarm.UpdateConfig{
				Monitor: time.Millisecond,
			},
		},
	}
}

func unit64Ptr(value uint64) *uint64 {
	return &value
}

func TestWaitOnServiceWithErrorBeforeTimeout(t *testing.T) {
	serviceID := "abcdef"
	timeout := 20 * time.Second
	client := &fakeClient{
		serviceInspectFunc: func(string, types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return replicastedService(), nil, nil
		},
		taskListFunc: taskListFuncWithErrorAfter(2),
	}

	opts := WaitOnServiceOptions{Timeout: &timeout}
	err := WaitOnService(context.Background(), test.NewFakeCli(client), serviceID, opts)
	assert.EqualError(t, err, "failed to get tasks")
}

func taskListFuncWithErrorAfter(count int) func(options types.TaskListOptions) ([]swarm.Task, error) {
	counter := -1
	return func(options types.TaskListOptions) ([]swarm.Task, error) {
		counter++
		if counter == count {
			return nil, errors.New("failed to get tasks")
		}
		return nil, nil
	}
}

func TestWaitOnServiceWithAPIErrorNoTimeout(t *testing.T) {
	serviceID := "abcdef"
	client := &fakeClient{
		serviceInspectFunc: func(string, types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return replicastedService(), nil, nil
		},
		taskListFunc: taskListFuncWithErrorAfter(2),
	}

	opts := WaitOnServiceOptions{}
	err := WaitOnService(context.Background(), test.NewFakeCli(client), serviceID, opts)
	assert.EqualError(t, err, "failed to get tasks")
}

func TestWaitOnServiceSuccess(t *testing.T) {
	serviceID := "abcdef"
	counter := -1
	client := &fakeClient{
		serviceInspectFunc: func(string, types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return replicastedService(), nil, nil
		},
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			counter++
			if counter != 2 {
				return nil, nil
			}
			return runningTasks(5), nil
		},
	}

	cli := test.NewFakeCli(client)
	opts := WaitOnServiceOptions{}
	err := WaitOnService(context.Background(), cli, serviceID, opts)
	require.NoError(t, err)
	assert.Equal(t, "", cli.ErrBuffer().String())
	assert.Contains(t, cli.OutBuffer().String(), "overall progress: 5 out of 5 tasks")
}

func runningTasks(count int) []swarm.Task {
	tasks := []swarm.Task{}
	for i := 0; i < count; i++ {
		tasks = append(tasks, swarm.Task{
			Slot:         i,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
			DesiredState: swarm.TaskStateRunning,
		})
	}
	return tasks
}
