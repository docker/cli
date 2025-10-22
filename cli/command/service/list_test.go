package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestServiceListOrder(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(ctx context.Context, options client.ServiceListOptions) (client.ServiceListResult, error) {
			return client.ServiceListResult{
				Items: []swarm.Service{
					newService("a57dbe8", "service-1-foo"),
					newService("a57dbdd", "service-10-foo"),
					newService("aaaaaaa", "service-2-foo"),
				},
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.SetArgs([]string{})
	assert.Check(t, cmd.Flags().Set("format", "{{.Name}}"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "service-list-sort.golden")
}

// TestServiceListServiceStatus tests that the ServiceStatus struct is correctly
// propagated. For older API versions, the ServiceStatus is calculated locally,
// based on the tasks that are present in the swarm, and the nodes that they are
// running on.
// If the list command is ran with `--quiet` option, no attempt should be done to
// propagate the ServiceStatus struct if not present, and it should be set to an
// empty struct.
func TestServiceListServiceStatus(t *testing.T) {
	type listResponse struct {
		ID       string
		Replicas string
	}

	type testCase struct {
		doc       string
		withQuiet bool
		opts      clusterOpts
		cluster   *cluster
		expected  []listResponse
	}

	tests := []testCase{
		{
			// Getting no nodes, services or tasks back from the daemon should
			// not cause any problems
			doc:      "empty cluster",
			cluster:  &cluster{}, // force an empty cluster
			expected: []listResponse{},
		},
		{
			// Services are running, but no active nodes were found. On API v1.41
			// and up, the ServiceStatus is sent by the daemon, so this should not
			// affect the results.
			doc: "no active nodes",
			opts: clusterOpts{
				activeNodes:  0,
				runningTasks: 2,
				desiredTasks: 4,
			},
			expected: []listResponse{
				{ID: "replicated", Replicas: "2/4"},
				{ID: "global", Replicas: "0/0"},
				{ID: "none-id", Replicas: "0/0"},
			},
		},
		{
			doc: "active nodes, 1 task running",
			opts: clusterOpts{
				activeNodes:  3,
				runningTasks: 1,
				desiredTasks: 2,
			},
			expected: []listResponse{
				{ID: "replicated", Replicas: "1/2"},
				{ID: "global", Replicas: "1/3"},
				{ID: "none-id", Replicas: "0/0"},
			},
		},
		{
			doc: "active nodes, all tasks running",
			opts: clusterOpts{
				activeNodes:  3,
				runningTasks: 3,
				desiredTasks: 3,
			},
			expected: []listResponse{
				{ID: "replicated", Replicas: "3/3"},
				{ID: "global", Replicas: "3/3"},
				{ID: "none-id", Replicas: "0/0"},
			},
		},
	}

	matrix := make([]testCase, 0)
	for _, quiet := range []bool{false, true} {
		for _, tc := range tests {
			if quiet {
				tc.withQuiet = quiet
				tc.doc += " with quiet"
			}
			matrix = append(matrix, tc)
		}
	}

	for _, tc := range matrix {
		t.Run(tc.doc, func(t *testing.T) {
			if tc.cluster == nil {
				tc.cluster = generateCluster(t, tc.opts)
			}
			cli := test.NewFakeCli(&fakeClient{
				serviceListFunc: func(ctx context.Context, options client.ServiceListOptions) (client.ServiceListResult, error) {
					if !options.Status {
						// Don't return "ServiceStatus" if not requested, or on older API versions
						for i := range tc.cluster.services.Items {
							tc.cluster.services.Items[i].ServiceStatus = nil
						}
					}
					return tc.cluster.services, nil
				},
				taskListFunc: func(context.Context, client.TaskListOptions) (client.TaskListResult, error) {
					return tc.cluster.tasks, nil
				},
				nodeListFunc: func(ctx context.Context, options client.NodeListOptions) (client.NodeListResult, error) {
					return tc.cluster.nodes, nil
				},
			})
			cmd := newListCommand(cli)
			cmd.SetArgs([]string{})
			if tc.withQuiet {
				cmd.SetArgs([]string{"--quiet"})
			}
			_ = cmd.Flags().Set("format", "{{ json .}}")
			assert.NilError(t, cmd.Execute())

			lines := strings.Split(strings.TrimSpace(cli.OutBuffer().String()), "\n")
			jsonArr := fmt.Sprintf("[%s]", strings.Join(lines, ","))
			results := make([]listResponse, 0)
			assert.NilError(t, json.Unmarshal([]byte(jsonArr), &results))

			if tc.withQuiet {
				// With "quiet" enabled, ServiceStatus should not be propagated
				for i := range tc.expected {
					tc.expected[i].Replicas = "0/0"
				}
			}
			assert.Check(t, is.DeepEqual(tc.expected, results), "%+v", results)
		})
	}
}

type clusterOpts struct {
	activeNodes  uint64
	desiredTasks uint64
	runningTasks uint64
}

type cluster struct {
	services client.ServiceListResult
	tasks    client.TaskListResult
	nodes    client.NodeListResult
}

func generateCluster(t *testing.T, opts clusterOpts) *cluster {
	t.Helper()
	c := cluster{
		services: generateServices(t, opts),
		nodes:    generateNodes(t, opts.activeNodes),
	}
	c.tasks = generateTasks(t, c.services, c.nodes, opts)
	return &c
}

func generateServices(t *testing.T, opts clusterOpts) client.ServiceListResult {
	t.Helper()

	// Can't have more global tasks than nodes
	globalTasks := opts.runningTasks
	if globalTasks > opts.activeNodes {
		globalTasks = opts.activeNodes
	}
	return client.ServiceListResult{
		Items: []swarm.Service{
			*builders.Service(
				builders.ServiceID("replicated"),
				builders.ServiceName("01-replicated-service"),
				builders.ReplicatedService(opts.desiredTasks),
				builders.ServiceStatus(opts.desiredTasks, opts.runningTasks),
			),
			*builders.Service(
				builders.ServiceID("global"),
				builders.ServiceName("02-global-service"),
				builders.GlobalService(),
				builders.ServiceStatus(opts.activeNodes, globalTasks),
			),
			*builders.Service(
				builders.ServiceID("none-id"),
				builders.ServiceName("03-none-service"),
			),
		},
	}
}

func generateTasks(t *testing.T, services client.ServiceListResult, nodes client.NodeListResult, opts clusterOpts) client.TaskListResult {
	t.Helper()
	tasks := client.TaskListResult{}

	for _, s := range services.Items {
		if s.Spec.Mode.Replicated == nil && s.Spec.Mode.Global == nil {
			continue
		}
		var runningTasks, failedTasks, desiredTasks uint64

		// Set the number of desired tasks to generate, based on the service's mode
		if s.Spec.Mode.Replicated != nil {
			desiredTasks = *s.Spec.Mode.Replicated.Replicas
		} else if s.Spec.Mode.Global != nil {
			desiredTasks = opts.activeNodes
		}

		for _, n := range nodes.Items {
			if runningTasks < opts.runningTasks && n.Status.State != swarm.NodeStateDown {
				tasks.Items = append(tasks.Items, swarm.Task{
					NodeID:       n.ID,
					ServiceID:    s.ID,
					Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
					DesiredState: swarm.TaskStateRunning,
				})
				runningTasks++
			}

			// If the number of "running" tasks is lower than the desired number
			// of tasks of the service, fill in the remaining number of tasks
			// with failed tasks. These tasks have a desired "running" state,
			// and thus will be included when calculating the "desired" tasks
			// for services.
			if failedTasks < (desiredTasks - opts.runningTasks) {
				tasks.Items = append(tasks.Items, swarm.Task{
					NodeID:       n.ID,
					ServiceID:    s.ID,
					Status:       swarm.TaskStatus{State: swarm.TaskStateFailed},
					DesiredState: swarm.TaskStateRunning,
				})
				failedTasks++
			}

			// Also add tasks with DesiredState: Shutdown. These should not be
			// counted as running or desired tasks.
			tasks.Items = append(tasks.Items, swarm.Task{
				NodeID:       n.ID,
				ServiceID:    s.ID,
				Status:       swarm.TaskStatus{State: swarm.TaskStateShutdown},
				DesiredState: swarm.TaskStateShutdown,
			})
		}
	}
	return tasks
}

// generateNodes generates a "nodes" endpoint API response with the requested
// number of "ready" nodes. In addition, a "down" node is generated.
func generateNodes(t *testing.T, activeNodes uint64) client.NodeListResult {
	t.Helper()
	nodes := client.NodeListResult{}
	var i uint64
	for i = 0; i < activeNodes; i++ {
		nodes.Items = append(nodes.Items, swarm.Node{
			ID:     fmt.Sprintf("node-ready-%d", i),
			Status: swarm.NodeStatus{State: swarm.NodeStateReady},
		})
		nodes.Items = append(nodes.Items, swarm.Node{
			ID:     fmt.Sprintf("node-down-%d", i),
			Status: swarm.NodeStatus{State: swarm.NodeStateDown},
		})
	}
	return nodes
}
