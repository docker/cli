package volume

import (
	"context"

	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client
	volumeCreateFunc  func(options client.VolumeCreateOptions) (client.VolumeCreateResult, error)
	volumeInspectFunc func(volumeID string) (client.VolumeInspectResult, error)
	volumeListFunc    func(client.VolumeListOptions) (client.VolumeListResult, error)
	volumeRemoveFunc  func(volumeID string, force bool) error
	volumePruneFunc   func(opts client.VolumePruneOptions) (client.VolumePruneResult, error)
}

func (c *fakeClient) VolumeCreate(_ context.Context, options client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
	if c.volumeCreateFunc != nil {
		return c.volumeCreateFunc(options)
	}
	return client.VolumeCreateResult{}, nil
}

func (c *fakeClient) VolumeInspect(_ context.Context, volumeID string, options client.VolumeInspectOptions) (client.VolumeInspectResult, error) {
	if c.volumeInspectFunc != nil {
		return c.volumeInspectFunc(volumeID)
	}
	return client.VolumeInspectResult{}, nil
}

func (c *fakeClient) VolumeList(_ context.Context, options client.VolumeListOptions) (client.VolumeListResult, error) {
	if c.volumeListFunc != nil {
		return c.volumeListFunc(options)
	}
	return client.VolumeListResult{}, nil
}

func (c *fakeClient) VolumePrune(_ context.Context, opts client.VolumePruneOptions) (client.VolumePruneResult, error) {
	if c.volumePruneFunc != nil {
		return c.volumePruneFunc(opts)
	}
	return client.VolumePruneResult{}, nil
}

func (c *fakeClient) VolumeRemove(_ context.Context, volumeID string, options client.VolumeRemoveOptions) (client.VolumeRemoveResult, error) {
	if c.volumeRemoveFunc != nil {
		return client.VolumeRemoveResult{}, c.volumeRemoveFunc(volumeID, options.Force)
	}
	return client.VolumeRemoveResult{}, nil
}
