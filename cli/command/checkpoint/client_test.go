package checkpoint

import (
	"context"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	checkpointCreateFunc func(container string, options client.CheckpointCreateOptions) (client.CheckpointCreateResult, error)
	checkpointDeleteFunc func(container string, options client.CheckpointRemoveOptions) (client.CheckpointRemoveResult, error)
	checkpointListFunc   func(container string, options client.CheckpointListOptions) (client.CheckpointListResult, error)
}

func (cli *fakeClient) CheckpointCreate(_ context.Context, container string, options client.CheckpointCreateOptions) (client.CheckpointCreateResult, error) {
	if cli.checkpointCreateFunc != nil {
		return cli.checkpointCreateFunc(container, options)
	}
	return client.CheckpointCreateResult{}, nil
}

func (cli *fakeClient) CheckpointRemove(_ context.Context, container string, options client.CheckpointRemoveOptions) (client.CheckpointRemoveResult, error) {
	if cli.checkpointDeleteFunc != nil {
		return cli.checkpointDeleteFunc(container, options)
	}
	return client.CheckpointRemoveResult{}, nil
}

func (cli *fakeClient) CheckpointList(_ context.Context, container string, options client.CheckpointListOptions) (client.CheckpointListResult, error) {
	if cli.checkpointListFunc != nil {
		return cli.checkpointListFunc(container, options)
	}
	return client.CheckpointListResult{}, nil
}
