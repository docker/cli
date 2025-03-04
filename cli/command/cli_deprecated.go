package command

import (
	"path/filepath"

	"github.com/docker/cli/cli/config"
	manifeststore "github.com/docker/cli/cli/manifest/store"
	"github.com/docker/cli/cli/trust"
	notaryclient "github.com/theupdateframework/notary/client"
)

type DeprecatedNotaryClient interface {
	// NotaryClient provides a Notary Repository to interact with signed metadata for an image
	//
	// Deprecated: use [trust.GetNotaryRepository] instead. This method is no longer used and will be removed in the next release.
	NotaryClient(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error)
}

type DeprecatedManifestClient interface {
	// ManifestStore returns a store for local manifests
	//
	// Deprecated: use [manifeststore.NewStore] instead. This method is no longer used and will be removed in the next release.
	ManifestStore() manifeststore.Store
}

// NotaryClient provides a Notary Repository to interact with signed metadata for an image
func (cli *DockerCli) NotaryClient(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error) {
	return trust.GetNotaryRepository(cli.In(), cli.Out(), UserAgent(), imgRefAndAuth.RepoInfo(), imgRefAndAuth.AuthConfig(), actions...)
}

// ManifestStore returns a store for local manifests
//
// Deprecated: use [manifeststore.NewStore] instead. This method is no longer used and will be removed in the next release.
func (*DockerCli) ManifestStore() manifeststore.Store {
	return manifeststore.NewStore(filepath.Join(config.Dir(), "manifests"))
}
