package manifest

import (
	"context"

	"github.com/containerd/errdefs"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/manifest/types"
)

func normalizeReference(ref string) (reference.Named, error) {
	namedRef, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return nil, err
	}
	if _, isDigested := namedRef.(reference.Canonical); !isDigested {
		return reference.TagNameOnly(namedRef), nil
	}
	return namedRef, nil
}

// getManifest from the local store, and fallback to the remote registry if it
// doesn't exist locally
func getManifest(ctx context.Context, dockerCLI command.Cli, listRef, namedRef reference.Named, insecure bool) (types.ImageManifest, error) {
	data, err := newManifestStore(dockerCLI).Get(listRef, namedRef)
	switch {
	case errdefs.IsNotFound(err):
		return newRegistryClient(dockerCLI, insecure).GetManifest(ctx, namedRef)
	case err != nil:
		return types.ImageManifest{}, err
	case len(data.Raw) == 0:
		return newRegistryClient(dockerCLI, insecure).GetManifest(ctx, namedRef)
	default:
		return data, nil
	}
}
