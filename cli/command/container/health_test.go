package container

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// healthClient returns a fake client whose inspect returns the given states in
// order (repeating the last one once exhausted), and whose event-stream emits
// an event for every state after the first, so that the wait-loop advances.
func healthClient(states ...container.State) *fakeClient {
	inspectCount := 0
	return &fakeClient{
		inspectFunc: func(string) (client.ContainerInspectResult, error) {
			state := states[min(inspectCount, len(states)-1)]
			inspectCount++
			return client.ContainerInspectResult{
				Container: container.InspectResponse{ID: "container-id", State: &state},
			}, nil
		},
		eventsFunc: func(ctx context.Context, _ client.EventsListOptions) client.EventsResult {
			messages := make(chan events.Message, len(states))
			for range states {
				messages <- events.Message{Action: events.ActionHealthStatusHealthy}
			}
			return client.EventsResult{Messages: messages, Err: make(chan error)}
		},
	}
}

func runningWithHealth(status container.HealthStatus) container.State {
	return container.State{Running: true, Health: &container.Health{Status: status}}
}

func TestWaitForHealthy(t *testing.T) {
	for _, tc := range []struct {
		name        string
		states      []container.State
		expectedErr string
	}{
		{
			name:   "already healthy",
			states: []container.State{runningWithHealth(container.Healthy)},
		},
		{
			name: "healthy after starting",
			states: []container.State{
				runningWithHealth(container.Starting),
				runningWithHealth(container.Starting),
				runningWithHealth(container.Healthy),
			},
		},
		{
			name: "healthy after being unhealthy",
			states: []container.State{
				runningWithHealth(container.Unhealthy),
				runningWithHealth(container.Healthy),
			},
		},
		{
			name:        "without healthcheck",
			states:      []container.State{{Running: true}},
			expectedErr: "container container-id does not have a healthcheck configured",
		},
		{
			name: "container exits before becoming healthy",
			states: []container.State{
				runningWithHealth(container.Starting),
				{Running: false, ExitCode: 3, Health: &container.Health{Status: container.Unhealthy}},
			},
			expectedErr: "container container-id exited with code 3 before becoming healthy",
		},
		{
			name:        "restarting container keeps being waited for",
			states:      []container.State{{Restarting: true, Health: &container.Health{Status: container.Starting}}},
			expectedErr: "did not become healthy within",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := waitForHealthy(context.Background(), healthClient(tc.states...), 100*time.Millisecond, "container-id")
			if tc.expectedErr == "" {
				assert.NilError(t, err)
			} else {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			}
		})
	}
}

func TestWaitForHealthyTimeout(t *testing.T) {
	apiClient := healthClient(runningWithHealth(container.Starting))

	err := waitForHealthy(context.Background(), apiClient, 100*time.Millisecond, "container-id")
	assert.Check(t, is.Error(err, "container container-id did not become healthy within 100ms (health-status: starting); the container is left running"))
}

// TestWaitForHealthyContainerNotStopped verifies that timing out does not stop
// or remove the container.
func TestWaitForHealthyContainerNotStopped(t *testing.T) {
	apiClient := healthClient(runningWithHealth(container.Starting))
	apiClient.containerStopFunc = func(context.Context, string, client.ContainerStopOptions) (client.ContainerStopResult, error) {
		t.Error("container should not be stopped on timeout")
		return client.ContainerStopResult{}, nil
	}
	apiClient.containerKillFunc = func(context.Context, string, client.ContainerKillOptions) (client.ContainerKillResult, error) {
		t.Error("container should not be killed on timeout")
		return client.ContainerKillResult{}, nil
	}
	apiClient.containerRemoveFunc = func(context.Context, string, client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
		t.Error("container should not be removed on timeout")
		return client.ContainerRemoveResult{}, nil
	}

	err := waitForHealthy(context.Background(), apiClient, 100*time.Millisecond, "container-id")
	assert.Check(t, is.ErrorContains(err, "did not become healthy"))
}

// TestWaitForHealthyMultipleContainers verifies that the timeout is shared by
// all containers, and that all of them are waited for.
func TestWaitForHealthyMultipleContainers(t *testing.T) {
	var inspected []string
	apiClient := &fakeClient{
		inspectFunc: func(containerID string) (client.ContainerInspectResult, error) {
			inspected = append(inspected, containerID)
			state := runningWithHealth(container.Healthy)
			return client.ContainerInspectResult{
				Container: container.InspectResponse{ID: containerID, State: &state},
			}, nil
		},
	}

	err := waitForHealthy(context.Background(), apiClient, time.Minute, "one", "two", "three")
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(inspected, []string{"one", "two", "three"}))
}

func TestWaitForHealthyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForHealthy(ctx, healthClient(runningWithHealth(container.Starting)), time.Minute, "container-id")
	assert.Check(t, is.ErrorIs(err, context.Canceled))
}

func TestWaitForHealthyEventStreamError(t *testing.T) {
	state := runningWithHealth(container.Starting)
	apiClient := &fakeClient{
		inspectFunc: func(string) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{ID: "container-id", State: &state},
			}, nil
		},
		eventsFunc: func(context.Context, client.EventsListOptions) client.EventsResult {
			errs := make(chan error, 1)
			errs <- errors.New("connection reset")
			return client.EventsResult{Messages: make(chan events.Message), Err: errs}
		},
	}

	err := waitForHealthy(context.Background(), apiClient, time.Minute, "container-id")
	assert.Check(t, is.Error(err, "failed to wait for container container-id to become healthy: connection reset"))
}
