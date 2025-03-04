package command

import (
	"github.com/docker/cli/cli/trust"
	notaryclient "github.com/theupdateframework/notary/client"
)

type DeprecatedNotaryClient interface {
	// NotaryClient provides a Notary Repository to interact with signed metadata for an image
	//
	// Deprecated: use [trust.GetNotaryRepository] instead. This method is no longer used and will be removed in the next release.
	NotaryClient(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error)
}

// NotaryClient provides a Notary Repository to interact with signed metadata for an image
func (cli *DockerCli) NotaryClient(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error) {
	return trust.GetNotaryRepository(cli.In(), cli.Out(), UserAgent(), imgRefAndAuth.RepoInfo(), imgRefAndAuth.AuthConfig(), actions...)
}
