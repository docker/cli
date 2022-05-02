package volume

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client
	volumeCreateFunc  func(volume.CreateOptions) (volume.Volume, error)
	volumeInspectFunc func(volumeID string) (volume.Volume, error)
	volumeListFunc    func(filter filters.Args) (volume.ListResponse, error)
	volumeRemoveFunc  func(volumeID string, force bool) error
	volumePruneFunc   func(filter filters.Args) (types.VolumesPruneReport, error)
}

func (c *fakeClient) VolumeCreate(_ context.Context, options volume.CreateOptions) (volume.Volume, error) {
	if c.volumeCreateFunc != nil {
		return c.volumeCreateFunc(options)
	}
	return volume.Volume{}, nil
}

func (c *fakeClient) VolumeInspect(_ context.Context, volumeID string) (volume.Volume, error) {
	if c.volumeInspectFunc != nil {
		return c.volumeInspectFunc(volumeID)
	}
	return volume.Volume{}, nil
}

func (c *fakeClient) VolumeList(_ context.Context, filter filters.Args) (volume.ListResponse, error) {
	if c.volumeListFunc != nil {
		return c.volumeListFunc(filter)
	}
	return volume.ListResponse{}, nil
}

func (c *fakeClient) VolumesPrune(_ context.Context, filter filters.Args) (types.VolumesPruneReport, error) {
	if c.volumePruneFunc != nil {
		return c.volumePruneFunc(filter)
	}
	return types.VolumesPruneReport{}, nil
}

func (c *fakeClient) VolumeRemove(_ context.Context, volumeID string, force bool) error {
	if c.volumeRemoveFunc != nil {
		return c.volumeRemoveFunc(volumeID, force)
	}
	return nil
}
