package command

import (
	"context"

	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/docker/api/types/registry"
)

type DeprecatedManifestClient interface {
	// RegistryClient returns a client for communicating with a Docker distribution
	// registry.
	//
	// Deprecated: use [registryclient.NewRegistryClient]. This method is no longer used and will be removed in the next release.
	RegistryClient(bool) registryclient.RegistryClient
}

// RegistryClient returns a client for communicating with a Docker distribution
// registry
//
// Deprecated: use [registryclient.NewRegistryClient]. This method is no longer used and will be removed in the next release.
func (cli *DockerCli) RegistryClient(allowInsecure bool) registryclient.RegistryClient {
	resolver := func(ctx context.Context, index *registry.IndexInfo) registry.AuthConfig {
		return ResolveAuthConfig(cli.ConfigFile(), index)
	}
	return registryclient.NewRegistryClient(resolver, UserAgent(), allowInsecure)
}
