package command

import (
	"context"
	"path/filepath"

	"github.com/docker/cli/cli/config"
	manifeststore "github.com/docker/cli/cli/manifest/store"
	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/docker/api/types/registry"
)

type DeprecatedManifestClient interface {
	// ManifestStore returns a store for local manifests
	//
	// Deprecated: use [manifeststore.NewStore] instead. This method is no longer used and will be removed in the next release.
	ManifestStore() manifeststore.Store

	// RegistryClient returns a client for communicating with a Docker distribution
	// registry.
	//
	// Deprecated: use [registryclient.NewRegistryClient]. This method is no longer used and will be removed in the next release.
	RegistryClient(bool) registryclient.RegistryClient
}

// ManifestStore returns a store for local manifests
//
// Deprecated: use [manifeststore.NewStore] instead. This method is no longer used and will be removed in the next release.
func (*DockerCli) ManifestStore() manifeststore.Store {
	return manifeststore.NewStore(filepath.Join(config.Dir(), "manifests"))
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
