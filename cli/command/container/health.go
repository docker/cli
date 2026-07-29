package container

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

// waitForHealthy blocks until all the given containers report a "healthy"
// health-status, or until timeout has passed, whichever comes first.
//
// Containers that fail to become healthy in time are left running: it's up to
// the user to decide whether to inspect, stop, or remove them.
func waitForHealthy(ctx context.Context, apiClient client.APIClient, timeout time.Duration, containerIDs ...string) error {
	// A single deadline is shared by all containers; waiting for one container
	// does not prevent the others from becoming healthy in the meantime, so
	// they can be waited for in sequence.
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for _, containerID := range containerIDs {
		if err := waitContainerHealthy(ctx, apiClient, containerID, timeout); err != nil {
			return err
		}
	}
	return nil
}

func waitContainerHealthy(ctx context.Context, apiClient client.APIClient, containerID string, timeout time.Duration) error {
	// Subscribe to the container's events before looking at its current state,
	// so that a health-status transition happening in between cannot be missed.
	eventRes := apiClient.Events(ctx, client.EventsListOptions{
		Filters: make(client.Filters).
			Add("type", string(events.ContainerEventType)).
			Add("container", containerID),
	})

	// The health-status in "health_status" events is embedded in the event's
	// action as free-form text, so the container is inspected on every event
	// instead; the daemon is the source of truth for the current status.
	for {
		state, err := inspectHealth(ctx, apiClient, containerID)
		if err != nil {
			return err
		}
		switch {
		case state.status == container.Healthy:
			return nil
		case state.status == container.NoHealthcheck:
			return fmt.Errorf("container %s does not have a healthcheck configured", containerID)
		case !state.running:
			return fmt.Errorf("container %s exited with code %d before becoming healthy", containerID, state.exitCode)
		}

		select {
		case <-ctx.Done():
			return healthWaitError(ctx, containerID, state.status, timeout)
		case <-eventRes.Messages:
		case err := <-eventRes.Err:
			if ctx.Err() != nil {
				return healthWaitError(ctx, containerID, state.status, timeout)
			}
			return fmt.Errorf("failed to wait for container %s to become healthy: %w", containerID, err)
		}
	}
}

// healthWaitError produces the error to return when waiting was interrupted;
// either because the timeout passed, or because the context was canceled.
func healthWaitError(ctx context.Context, containerID string, status container.HealthStatus, timeout time.Duration) error {
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ctx.Err()
	}
	return fmt.Errorf("container %s did not become healthy within %s (health-status: %s); the container is left running", containerID, timeout, status)
}

// healthState is a snapshot of the parts of a container's state that are
// relevant when waiting for it to become healthy.
type healthState struct {
	status   container.HealthStatus
	running  bool
	exitCode int
}

func inspectHealth(ctx context.Context, apiClient client.APIClient, containerID string) (healthState, error) {
	res, err := apiClient.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return healthState{}, err
	}
	if res.Container.State == nil {
		return healthState{}, fmt.Errorf("container %s has no state", containerID)
	}
	state := healthState{
		status: container.NoHealthcheck,
		// A restarting container may still become healthy, so it's not
		// considered to have stopped.
		running:  res.Container.State.Running || res.Container.State.Restarting,
		exitCode: res.Container.State.ExitCode,
	}
	if res.Container.State.Health != nil {
		state.status = res.Container.State.Health.Status
	}
	return state, nil
}
