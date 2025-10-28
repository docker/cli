package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNodePsErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		infoFunc        func() (client.SystemInfoResult, error)
		nodeInspectFunc func() (client.NodeInspectResult, error)
		taskListFunc    func(options client.TaskListOptions) (client.TaskListResult, error)
		expectedError   string
	}{
		{
			infoFunc: func() (client.SystemInfoResult, error) {
				return client.SystemInfoResult{}, errors.New("error asking for node info")
			},
			expectedError: "error asking for node info",
		},
		{
			args: []string{"nodeID"},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{}, errors.New("error inspecting the node")
			},
			expectedError: "error inspecting the node",
		},
		{
			args: []string{"nodeID"},
			taskListFunc: func(options client.TaskListOptions) (client.TaskListResult, error) {
				return client.TaskListResult{}, errors.New("error returning the task list")
			},
			expectedError: "error returning the task list",
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			infoFunc:        tc.infoFunc,
			nodeInspectFunc: tc.nodeInspectFunc,
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
		nodeInspectFunc    func() (client.NodeInspectResult, error)
		taskListFunc       func(options client.TaskListOptions) (client.TaskListResult, error)
		taskInspectFunc    func(taskID string) (client.TaskInspectResult, error)
		serviceInspectFunc func(ctx context.Context, serviceID string, opts client.ServiceInspectOptions) (client.ServiceInspectResult, error)
	}{
		{
			name: "simple",
			args: []string{"nodeID"},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(),
				}, nil
			},
			taskListFunc: func(options client.TaskListOptions) (client.TaskListResult, error) {
				return client.TaskListResult{
					Items: []swarm.Task{
						*builders.Task(builders.WithStatus(builders.Timestamp(time.Now().Add(-2*time.Hour)), builders.PortStatus([]swarm.PortConfig{
							{
								TargetPort:    80,
								PublishedPort: 80,
								Protocol:      "tcp",
							},
						}))),
					},
				}, nil
			},
			serviceInspectFunc: func(ctx context.Context, serviceID string, opts client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
				return client.ServiceInspectResult{
					Service: swarm.Service{
						ID: serviceID,
						Spec: swarm.ServiceSpec{
							Annotations: swarm.Annotations{
								Name: serviceID,
							},
						},
					},
				}, nil
			},
		},
		{
			name: "with-errors",
			args: []string{"nodeID"},
			nodeInspectFunc: func() (client.NodeInspectResult, error) {
				return client.NodeInspectResult{
					Node: *builders.Node(),
				}, nil
			},
			taskListFunc: func(options client.TaskListOptions) (client.TaskListResult, error) {
				return client.TaskListResult{
					Items: []swarm.Task{
						*builders.Task(builders.TaskID("taskID1"), builders.TaskServiceID("failure"),
							builders.WithStatus(builders.Timestamp(time.Now().Add(-2*time.Hour)), builders.StatusErr("a task error"))),
						*builders.Task(builders.TaskID("taskID2"), builders.TaskServiceID("failure"),
							builders.WithStatus(builders.Timestamp(time.Now().Add(-3*time.Hour)), builders.StatusErr("a task error"))),
						*builders.Task(builders.TaskID("taskID3"), builders.TaskServiceID("failure"),
							builders.WithStatus(builders.Timestamp(time.Now().Add(-4*time.Hour)), builders.StatusErr("a task error"))),
					},
				}, nil
			},
			serviceInspectFunc: func(ctx context.Context, serviceID string, opts client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
				return client.ServiceInspectResult{
					Service: swarm.Service{
						ID: serviceID,
						Spec: swarm.ServiceSpec{
							Annotations: swarm.Annotations{
								Name: serviceID,
							},
						},
					},
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
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
