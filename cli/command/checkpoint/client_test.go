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

func (cli *fakeClient) CheckpointCreate(ctx context.Context, container string, options types.CheckpointCreateOptions) error {
	if cli.checkpointCreateFunc != nil {
		return cli.checkpointCreateFunc(container, options)
	}
	return nil
}

func (cli *fakeClient) CheckpointDelete(ctx context.Context, container string, options types.CheckpointDeleteOptions) error {
	if cli.checkpointDeleteFunc != nil {
		return cli.checkpointDeleteFunc(container, options)
	}
	return nil
}

func (cli *fakeClient) CheckpointList(ctx context.Context, container string, options types.CheckpointListOptions) ([]types.Checkpoint, error) {
	if cli.checkpointListFunc != nil {
		return cli.checkpointListFunc(container, options)
	}
	return []types.Checkpoint{}, nil
}
