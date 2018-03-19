package engine

import (
	"context"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/pkg/errors"
)

func getRegistryAuth(cli command.Cli, registryPrefix string) (*types.AuthConfig, error) {
	if registryPrefix == "" {
		registryPrefix = "docker.io/docker"
	}
	distributionRef, err := reference.ParseNormalizedNamed(registryPrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse image name: %s", registryPrefix)
	}
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(context.Background(), nil, authResolver(cli), distributionRef.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get imgRefAndAuth")
	}
	return imgRefAndAuth.AuthConfig(), nil
}

func authResolver(cli command.Cli) func(ctx context.Context, index *registrytypes.IndexInfo) types.AuthConfig {
	return func(ctx context.Context, index *registrytypes.IndexInfo) types.AuthConfig {
		return command.ResolveAuthConfig(ctx, cli, index)
	}
}
