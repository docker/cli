package volume

import (
	"context"

	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	volumeCreateFunc  func(volume.CreateOptions) (volume.Volume, error)
	volumeInspectFunc func(volumeID string) (volume.Volume, error)
	volumeListFunc    func(filter client.Filters) (volume.ListResponse, error)
	volumeRemoveFunc  func(volumeID string, force bool) error
	volumePruneFunc   func(filter client.Filters) (volume.PruneReport, error)
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

func (c *fakeClient) VolumeList(_ context.Context, options client.VolumeListOptions) (volume.ListResponse, error) {
	if c.volumeListFunc != nil {
		return c.volumeListFunc(options.Filters)
	}
	return volume.ListResponse{}, nil
}

func (c *fakeClient) VolumesPrune(_ context.Context, filter client.Filters) (volume.PruneReport, error) {
	if c.volumePruneFunc != nil {
		return c.volumePruneFunc(filter)
	}
	return volume.PruneReport{}, nil
}

func (c *fakeClient) VolumeRemove(_ context.Context, volumeID string, force bool) error {
	if c.volumeRemoveFunc != nil {
		return c.volumeRemoveFunc(volumeID, force)
	}
	return nil
}
