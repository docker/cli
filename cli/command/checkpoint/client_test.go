package checkpoint

import (
	"context"

	"github.com/moby/moby/api/types/checkpoint"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	checkpointCreateFunc func(container string, options client.CheckpointCreateOptions) error
	checkpointDeleteFunc func(container string, options client.CheckpointDeleteOptions) error
	checkpointListFunc   func(container string, options client.CheckpointListOptions) ([]checkpoint.Summary, error)
}

func (cli *fakeClient) CheckpointCreate(_ context.Context, container string, options client.CheckpointCreateOptions) error {
	if cli.checkpointCreateFunc != nil {
		return cli.checkpointCreateFunc(container, options)
	}
	return nil
}

func (cli *fakeClient) CheckpointDelete(_ context.Context, container string, options client.CheckpointDeleteOptions) error {
	if cli.checkpointDeleteFunc != nil {
		return cli.checkpointDeleteFunc(container, options)
	}
	return nil
}

func (cli *fakeClient) CheckpointList(_ context.Context, container string, options client.CheckpointListOptions) ([]checkpoint.Summary, error) {
	if cli.checkpointListFunc != nil {
		return cli.checkpointListFunc(container, options)
	}
	return []checkpoint.Summary{}, nil
}
