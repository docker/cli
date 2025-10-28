package container

import (
	"context"
	"errors"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/sirupsen/logrus"
)

func waitExitOrRemoved(ctx context.Context, apiClient client.APIClient, containerID string, waitRemove bool) <-chan int {
	if len(containerID) == 0 {
		// containerID can never be empty
		panic("Internal Error: waitExitOrRemoved needs a containerID as parameter")
	}

	condition := container.WaitConditionNextExit
	if waitRemove {
		condition = container.WaitConditionRemoved
	}

	waitRes := apiClient.ContainerWait(ctx, containerID, client.ContainerWaitOptions{
		Condition: condition,
	})

	statusC := make(chan int)
	go func() {
		defer close(statusC)
		select {
		case <-ctx.Done():
			return
		case result := <-waitRes.Result:
			if result.Error != nil {
				logrus.Errorf("Error waiting for container: %v", result.Error.Message)
				statusC <- 125
			} else {
				statusC <- int(result.StatusCode)
			}
		case err := <-waitRes.Error:
			if errors.Is(err, context.Canceled) {
				return
			}
			logrus.Errorf("error waiting for container: %v", err)
			statusC <- 125
		}
	}()

	return statusC
}

func parallelOperation(ctx context.Context, containers []string, op func(ctx context.Context, containerID string) error) chan error {
	if len(containers) == 0 {
		return nil
	}
	const defaultParallel int = 50
	sem := make(chan struct{}, defaultParallel)
	errChan := make(chan error)

	// make sure result is printed in correct order
	output := map[string]chan error{}
	for _, c := range containers {
		output[c] = make(chan error, 1)
	}
	go func() {
		for _, c := range containers {
			err := <-output[c]
			errChan <- err
		}
	}()

	go func() {
		for _, c := range containers {
			sem <- struct{}{} // Wait for active queue sem to drain.
			go func(container string) {
				output[container] <- op(ctx, container)
				<-sem
			}(c)
		}
	}()
	return errChan
}
