package checkpoint

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	checkpointCreateFunc func(container string, options types.CheckpointCreateOptions) error
	checkpointDeleteFunc func(container string, options types.CheckpointDeleteOptions) error
	checkpointListFunc   func(container string, options types.CheckpointListOptions) ([]types.Checkpoint, error)
}

// CheckpointCreate creates a container checkpoint using provided options or delegates to the stored function.
func (cli *fakeClient) CheckpointCreate(_ context.Context, container string, options types.CheckpointCreateOptions) error {
	if cli.checkpointCreateFunc != nil {
		return cli.checkpointCreateFunc(container, options)
	}
	return nil
}

// CheckpointDelete deletes a container checkpoint based on provided options or delegates to the stored function.
func (cli *fakeClient) CheckpointDelete(_ context.Context, container string, options types.CheckpointDeleteOptions) error {
	if cli.checkpointDeleteFunc != nil {
		return cli.checkpointDeleteFunc(container, options)
	}
	return nil
}

// CheckpointList lists all container checkpoints based on provided options or delegates to the stored function.
func (cli *fakeClient) CheckpointList(_ context.Context, container string, options types.CheckpointListOptions) ([]types.Checkpoint, error) {
	if cli.checkpointListFunc != nil {
		return cli.checkpointListFunc(container, options)
	}
	return []types.Checkpoint{}, nil
}
