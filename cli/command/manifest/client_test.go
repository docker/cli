package manifest

import (
	manifesttypes "github.com/docker/cli/cli/manifest/types"
	"github.com/docker/cli/cli/registry/client"
	"github.com/docker/distribution/reference"
	"golang.org/x/net/context"
)

type fakeRegistryClient struct {
	client.RegistryClient
	getManifestFunc     func(ctx context.Context, ref reference.Named) (manifesttypes.ImageManifest, error)
	getManifestListFunc func(ctx context.Context, ref reference.Named) ([]manifesttypes.ImageManifest, error)
}

func (c *fakeRegistryClient) GetManifest(ctx context.Context, ref reference.Named) (manifesttypes.ImageManifest, error) {
	if c.getManifestFunc != nil {
		return c.getManifestFunc(ctx, ref)
	}
	return manifesttypes.ImageManifest{}, nil
}

func (c *fakeRegistryClient) GetManifestList(ctx context.Context, ref reference.Named) ([]manifesttypes.ImageManifest, error) {
	if c.getManifestListFunc != nil {
		return c.getManifestListFunc(ctx, ref)
	}
	return nil, nil
}
