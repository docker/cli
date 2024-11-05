package node

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNodePsErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		infoFunc        func() (system.Info, error)
		nodeInspectFunc func() (swarm.Node, []byte, error)
		taskListFunc    func(options types.TaskListOptions) ([]swarm.Task, error)
		taskInspectFunc func(taskID string) (swarm.Task, []byte, error)
		expectedError   string
	}{
		{
			infoFunc: func() (system.Info, error) {
				return system.Info{}, errors.Errorf("error asking for node info")
			},
			expectedError: "error asking for node info",
		},
		{
			args: []string{"nodeID"},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return swarm.Node{}, []byte{}, errors.Errorf("error inspecting the node")
			},
			expectedError: "error inspecting the node",
		},
		{
			args: []string{"nodeID"},
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{}, errors.Errorf("error returning the task list")
			},
			expectedError: "error returning the task list",
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			infoFunc:        tc.infoFunc,
			nodeInspectFunc: tc.nodeInspectFunc,
			taskInspectFunc: tc.taskInspectFunc,
			taskListFunc:    tc.taskListFunc,
		})
		cmd := newPsCommand(cli)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			assert.Check(t, cmd.Flags().Set(key, value))
		}
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.Error(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNodePs(t *testing.T) {
	testCases := []struct {
		name               string
		args               []string
		flags              map[string]string
		infoFunc           func() (system.Info, error)
		nodeInspectFunc    func() (swarm.Node, []byte, error)
		taskListFunc       func(options types.TaskListOptions) ([]swarm.Task, error)
		taskInspectFunc    func(taskID string) (swarm.Task, []byte, error)
		serviceInspectFunc func(ctx context.Context, serviceID string, opts types.ServiceInspectOptions) (swarm.Service, []byte, error)
	}{
		{
			name: "simple",
			args: []string{"nodeID"},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return *builders.Node(), []byte{}, nil
			},
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{
					*builders.Task(builders.WithStatus(builders.Timestamp(time.Now().Add(-2*time.Hour)), builders.PortStatus([]swarm.PortConfig{
						{
							TargetPort:    80,
							PublishedPort: 80,
							Protocol:      "tcp",
						},
					}))),
				}, nil
			},
			serviceInspectFunc: func(ctx context.Context, serviceID string, opts types.ServiceInspectOptions) (swarm.Service, []byte, error) {
				return swarm.Service{
					ID: serviceID,
					Spec: swarm.ServiceSpec{
						Annotations: swarm.Annotations{
							Name: serviceID,
						},
					},
				}, []byte{}, nil
			},
		},
		{
			name: "with-errors",
			args: []string{"nodeID"},
			nodeInspectFunc: func() (swarm.Node, []byte, error) {
				return *builders.Node(), []byte{}, nil
			},
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{
					*builders.Task(builders.TaskID("taskID1"), builders.TaskServiceID("failure"),
						builders.WithStatus(builders.Timestamp(time.Now().Add(-2*time.Hour)), builders.StatusErr("a task error"))),
					*builders.Task(builders.TaskID("taskID2"), builders.TaskServiceID("failure"),
						builders.WithStatus(builders.Timestamp(time.Now().Add(-3*time.Hour)), builders.StatusErr("a task error"))),
					*builders.Task(builders.TaskID("taskID3"), builders.TaskServiceID("failure"),
						builders.WithStatus(builders.Timestamp(time.Now().Add(-4*time.Hour)), builders.StatusErr("a task error"))),
				}, nil
			},
			serviceInspectFunc: func(ctx context.Context, serviceID string, opts types.ServiceInspectOptions) (swarm.Service, []byte, error) {
				return swarm.Service{
					ID: serviceID,
					Spec: swarm.ServiceSpec{
						Annotations: swarm.Annotations{
							Name: serviceID,
						},
					},
				}, []byte{}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				infoFunc:           tc.infoFunc,
				nodeInspectFunc:    tc.nodeInspectFunc,
				taskInspectFunc:    tc.taskInspectFunc,
				taskListFunc:       tc.taskListFunc,
				serviceInspectFunc: tc.serviceInspectFunc,
			})
			cmd := newPsCommand(cli)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				assert.Check(t, cmd.Flags().Set(key, value))
			}
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("node-ps.%s.golden", tc.name))
		})
	}
}
