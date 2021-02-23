package clustervolume

// client_test.go includes a fake implementation of the volumes client.

import (
	"context"
	"errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// fakeClient is the faked implementation of the docker client. it contains
// functions in its fields that are called when the corresponding method is
// called, allowing for injection.
type fakeClient struct {
	client.Client

	listFunc    func(ctx context.Context, options types.VolumeListOptions) ([]swarm.Volume, error)
	inspectFunc func(ctx context.Context, id string) (swarm.Volume, []byte, error)
	createFunc  func(ctx context.Context, volume swarm.VolumeSpec) (types.VolumeCreateResponse, error)
	updateFunc  func(ctx context.Context, id string, version swarm.Version, volume swarm.VolumeSpec) error
	removeFunc  func(ctx context.Context, id string) error
}

func (f *fakeClient) ClusterVolumeList(ctx context.Context, options types.VolumeListOptions) ([]swarm.Volume, error) {
	if f.listFunc != nil {
		return f.listFunc(ctx, options)
	}
	return nil, errors.New("listFunc not defined")
}

func (f *fakeClient) ClusterVolumeInspectWithRaw(ctx context.Context, id string) (swarm.Volume, []byte, error) {
	if f.inspectFunc != nil {
		return f.inspectFunc(ctx, id)
	}
	return swarm.Volume{}, nil, errors.New("inspectFunc not defined")
}

func (f *fakeClient) ClusterVolumeCreate(ctx context.Context, volume swarm.VolumeSpec) (types.VolumeCreateResponse, error) {
	if f.createFunc != nil {
		return f.createFunc(ctx, volume)
	}
	return types.VolumeCreateResponse{}, errors.New("createFunc not defined")
}

func (f *fakeClient) ClusterVolumeUpdate(ctx context.Context, id string, version swarm.Version, volume swarm.VolumeSpec) error {
	if f.updateFunc != nil {
		return f.updateFunc(ctx, id, version, volume)
	}
	return errors.New("updateFunc not defined")
}

func (f *fakeClient) ClusterVolumeRemove(ctx context.Context, id string) error {
	if f.removeFunc != nil {
		return f.removeFunc(ctx, id)
	}
	return errors.New("removeFunc not defined")
}
