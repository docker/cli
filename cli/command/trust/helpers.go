package trust

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	"github.com/docker/notary/client"
	"github.com/docker/notary/tuf/data"
)

const releasedRoleName = "Repo Admin"

// ImageRefAndAuth contains all reference information and the auth config for an image request
type ImageRefAndAuth struct {
	authConfig *types.AuthConfig
	reference  reference.Named
	repoInfo   *registry.RepositoryInfo
	tag        string
}

// AuthConfig returns the auth information (username, etc) for a given ImageRefAndAuth
func (imgRefAuth *ImageRefAndAuth) AuthConfig() *types.AuthConfig {
	return imgRefAuth.authConfig
}

// Reference returns the Image reference for a given ImageRefAndAuth
func (imgRefAuth *ImageRefAndAuth) Reference() reference.Named {
	return imgRefAuth.reference
}

// RepoInfo returns the repository information for a given ImageRefAndAuth
func (imgRefAuth *ImageRefAndAuth) RepoInfo() *registry.RepositoryInfo {
	return imgRefAuth.repoInfo
}

// Tag returns the Image tag for a given ImageRefAndAuth
func (imgRefAuth *ImageRefAndAuth) Tag() string {
	return imgRefAuth.tag
}

func getImageReferencesAndAuth(ctx context.Context, cli command.Cli, imgName string) (*ImageRefAndAuth, error) {
	ref, err := reference.ParseNormalizedNamed(imgName)
	if err != nil {
		return nil, err
	}

	tag, err := getTag(ref)
	if err != nil {
		return nil, err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return nil, err
	}

	authConfig := command.ResolveAuthConfig(ctx, cli, repoInfo.Index)
	return &ImageRefAndAuth{&authConfig, ref, repoInfo, tag}, err
}

func getTag(ref reference.Named) (string, error) {
	var tag string
	switch x := ref.(type) {
	case reference.Canonical:
		return "", fmt.Errorf("cannot use a digest reference for IMAGE:TAG")
	case reference.NamedTagged:
		tag = x.Tag()
	default:
		tag = ""
	}
	return tag, nil
}

// check if a role name is "released": either targets/releases or targets TUF roles
func isReleasedTarget(role data.RoleName) bool {
	return role == data.CanonicalTargetsRole || role == trust.ReleasesRole
}

// convert TUF role name to a human-understandable signer name
func notaryRoleToSigner(tufRole data.RoleName) string {
	//  don't show a signer for "targets" or "targets/releases"
	if isReleasedTarget(data.RoleName(tufRole.String())) {
		return releasedRoleName
	}
	return strings.TrimPrefix(tufRole.String(), "targets/")
}

func clearChangeList(notaryRepo client.Repository) error {
	cl, err := notaryRepo.GetChangelist()
	if err != nil {
		return err
	}
	return cl.Clear("")
}
