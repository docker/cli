package checkpoint

import (
	"context"

	"github.com/docker/docker/api/types/checkpoint"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	checkpointCreateFunc func(container string, options checkpoint.CreateOptions) error
	checkpointDeleteFunc func(container string, options checkpoint.DeleteOptions) error
	checkpointListFunc   func(container string, options checkpoint.ListOptions) ([]checkpoint.Summary, error)
}

func (cli *fakeClient) CheckpointCreate(_ context.Context, container string, options checkpoint.CreateOptions) error {
	if cli.checkpointCreateFunc != nil {
		return cli.checkpointCreateFunc(container, options)
	}
	return nil
}

func (cli *fakeClient) CheckpointDelete(_ context.Context, container string, options checkpoint.DeleteOptions) error {
	if cli.checkpointDeleteFunc != nil {
		return cli.checkpointDeleteFunc(container, options)
	}
	return nil
}

func (cli *fakeClient) CheckpointList(_ context.Context, container string, options checkpoint.ListOptions) ([]checkpoint.Summary, error) {
	if cli.checkpointListFunc != nil {
		return cli.checkpointListFunc(container, options)
	}
	return []checkpoint.Summary{}, nil
}
