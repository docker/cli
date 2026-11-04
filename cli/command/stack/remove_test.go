package stack

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func fakeClientForRemoveStackTest() *fakeClient {
	allServices := []string{
		objectName("foo", "service1"),
		objectName("foo", "service2"),
		objectName("bar", "service1"),
		objectName("bar", "service2"),
	}
	allNetworks := []string{
		objectName("foo", "network1"),
		objectName("bar", "network1"),
	}
	allSecrets := []string{
		objectName("foo", "secret1"),
		objectName("foo", "secret2"),
		objectName("bar", "secret1"),
	}
	allConfigs := []string{
		objectName("foo", "config1"),
		objectName("foo", "config2"),
		objectName("bar", "config1"),
	}
	return &fakeClient{
		services: allServices,
		networks: allNetworks,
		secrets:  allSecrets,
		configs:  allConfigs,
	}
}

func TestRemoveWithEmptyName(t *testing.T) {
	cmd := newRemoveCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetArgs([]string{"good", "'   '", "alsogood"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	assert.ErrorContains(t, cmd.Execute(), `invalid stack name: "'   '"`)
}

func TestRemoveStackRemovesEverything(t *testing.T) {
	apiClient := fakeClientForRemoveStackTest()
	cmd := newRemoveCommand(test.NewFakeCli(apiClient))
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.DeepEqual(buildObjectIDs(apiClient.services), apiClient.removedServices))
	assert.Check(t, is.DeepEqual(buildObjectIDs(apiClient.networks), apiClient.removedNetworks))
	assert.Check(t, is.DeepEqual(buildObjectIDs(apiClient.secrets), apiClient.removedSecrets))
	assert.Check(t, is.DeepEqual(buildObjectIDs(apiClient.configs), apiClient.removedConfigs))
}

func TestRemoveStackSkipEmpty(t *testing.T) {
	allServices := []string{objectName("bar", "service1"), objectName("bar", "service2")}
	allServiceIDs := buildObjectIDs(allServices)

	allNetworks := []string{objectName("bar", "network1")}
	allNetworkIDs := buildObjectIDs(allNetworks)

	allSecrets := []string{objectName("bar", "secret1")}
	allSecretIDs := buildObjectIDs(allSecrets)

	allConfigs := []string{objectName("bar", "config1")}
	allConfigIDs := buildObjectIDs(allConfigs)

	apiClient := &fakeClient{
		services: allServices,
		networks: allNetworks,
		secrets:  allSecrets,
		configs:  allConfigs,
	}
	fakeCli := test.NewFakeCli(apiClient)
	cmd := newRemoveCommand(fakeCli)
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NilError(t, cmd.Execute())
	expectedList := []string{
		"Removing service bar_service1",
		"Removing service bar_service2",
		"Removing secret bar_secret1",
		"Removing config bar_config1",
		"Removing network bar_network1\n",
	}
	assert.Check(t, is.Equal(strings.Join(expectedList, "\n"), fakeCli.OutBuffer().String()))
	assert.Check(t, is.Contains(fakeCli.ErrBuffer().String(), "Nothing found in stack: foo\n"))
	assert.Check(t, is.DeepEqual(allServiceIDs, apiClient.removedServices))
	assert.Check(t, is.DeepEqual(allNetworkIDs, apiClient.removedNetworks))
	assert.Check(t, is.DeepEqual(allSecretIDs, apiClient.removedSecrets))
	assert.Check(t, is.DeepEqual(allConfigIDs, apiClient.removedConfigs))
}

func TestRemoveContinueAfterError(t *testing.T) {
	allServices := []string{objectName("foo", "service1"), objectName("bar", "service1")}
	allServiceIDs := buildObjectIDs(allServices)

	allNetworks := []string{objectName("foo", "network1"), objectName("bar", "network1")}
	allNetworkIDs := buildObjectIDs(allNetworks)

	allSecrets := []string{objectName("foo", "secret1"), objectName("bar", "secret1")}
	allSecretIDs := buildObjectIDs(allSecrets)

	allConfigs := []string{objectName("foo", "config1"), objectName("bar", "config1")}
	allConfigIDs := buildObjectIDs(allConfigs)

	var removedServices []string
	apiClient := &fakeClient{
		services: allServices,
		networks: allNetworks,
		secrets:  allSecrets,
		configs:  allConfigs,

		serviceRemoveFunc: func(serviceID string) (client.ServiceRemoveResult, error) {
			removedServices = append(removedServices, serviceID)

			if strings.Contains(serviceID, "foo") {
				return client.ServiceRemoveResult{}, errors.New("")
			}
			return client.ServiceRemoveResult{}, nil
		},
	}
	cmd := newRemoveCommand(test.NewFakeCli(apiClient))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"foo", "bar"})

	assert.Error(t, cmd.Execute(), "failed to remove some resources from stack: foo")
	assert.Check(t, is.DeepEqual(allServiceIDs, removedServices))
	assert.Check(t, is.DeepEqual(allNetworkIDs, apiClient.removedNetworks))
	assert.Check(t, is.DeepEqual(allSecretIDs, apiClient.removedSecrets))
	assert.Check(t, is.DeepEqual(allConfigIDs, apiClient.removedConfigs))
}

// taskListStoppingAfter returns a TaskList stub reporting two tasks: one that
// has already stopped, and one that keeps running until the stub has been
// polled the given number of times. It counts the polls in *calls.
func taskListStoppingAfter(polls int, calls *int) func(client.TaskListOptions) (client.TaskListResult, error) {
	return func(client.TaskListOptions) (client.TaskListResult, error) {
		*calls++
		last := swarm.TaskStateRunning
		if *calls >= polls {
			last = swarm.TaskStateShutdown
		}
		return client.TaskListResult{Items: []swarm.Task{
			{Status: swarm.TaskStatus{State: swarm.TaskStateShutdown}},
			{Status: swarm.TaskStatus{State: last}},
		}}, nil
	}
}

func TestWaitOnTasksWaitsForAllTasks(t *testing.T) {
	const pollsUntilStopped = 4
	var taskListCalls int
	apiClient := &fakeClient{
		taskListFunc: taskListStoppingAfter(pollsUntilStopped, &taskListCalls),
	}

	assert.NilError(t, waitOnTasks(context.Background(), apiClient, "foo"))
	assert.Check(t, is.Equal(pollsUntilStopped, taskListCalls))
}

func TestWaitOnTasksReturnsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	apiClient := &fakeClient{
		taskListFunc: func(client.TaskListOptions) (client.TaskListResult, error) {
			cancel()
			return client.TaskListResult{Items: []swarm.Task{
				{Status: swarm.TaskStatus{State: swarm.TaskStateRunning}},
			}}, nil
		},
	}

	assert.ErrorIs(t, waitOnTasks(ctx, apiClient, "foo"), context.Canceled)
}

func TestRemoveStackWaitsForTasksBeforeRemovingNetworks(t *testing.T) {
	const pollsUntilStopped = 4
	var taskListCalls, taskListCallsAtNetworkRemoval int
	apiClient := &fakeClient{
		services:     []string{objectName("foo", "service1")},
		networks:     []string{objectName("foo", "network1")},
		taskListFunc: taskListStoppingAfter(pollsUntilStopped, &taskListCalls),
	}
	apiClient.networkRemoveFunc = func(networkID string) error {
		taskListCallsAtNetworkRemoval = taskListCalls
		apiClient.removedNetworks = append(apiClient.removedNetworks, networkID)
		return nil
	}
	cmd := newRemoveCommand(test.NewFakeCli(apiClient))
	cmd.SetArgs([]string{"--detach=false", "foo"})
	cmd.SetOut(io.Discard)

	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.DeepEqual(buildObjectIDs(apiClient.services), apiClient.removedServices))
	assert.Check(t, is.DeepEqual(buildObjectIDs(apiClient.networks), apiClient.removedNetworks))
	assert.Check(t, is.Equal(pollsUntilStopped, taskListCallsAtNetworkRemoval))
}

func TestRemoveStackDetachedDoesNotWaitOnTasks(t *testing.T) {
	apiClient := &fakeClient{
		services: []string{objectName("foo", "service1")},
		networks: []string{objectName("foo", "network1")},
		taskListFunc: func(client.TaskListOptions) (client.TaskListResult, error) {
			return client.TaskListResult{}, errors.New("tasks must not be listed when detached")
		},
	}
	cmd := newRemoveCommand(test.NewFakeCli(apiClient))
	cmd.SetArgs([]string{"foo"})
	cmd.SetOut(io.Discard)

	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.DeepEqual(buildObjectIDs(apiClient.networks), apiClient.removedNetworks))
}
