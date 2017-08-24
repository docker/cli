package trust

import (
	"context"
	"fmt"
	"io"
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

func checkLocalImageExistence(ctx context.Context, cli command.Cli, imageName string) error {
	_, _, err := cli.Client().ImageInspectWithRaw(ctx, imageName)
	return err
}

func getImageReferencesAndAuth(cli command.Cli, imgName string) (context.Context, reference.Named, *registry.RepositoryInfo, *types.AuthConfig, error) {
	ref, err := reference.ParseNormalizedNamed(imgName)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	ctx := context.Background()
	authConfig := command.ResolveAuthConfig(ctx, cli, repoInfo.Index)
	return ctx, ref, repoInfo, &authConfig, err
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

func askConfirm(input io.Reader) bool {
	var res string
	if _, err := fmt.Fscanln(input, &res); err != nil {
		return false
	}
	if strings.EqualFold(res, "y") || strings.EqualFold(res, "yes") {
		return true
	}
	return false
}

func clearChangeList(notaryRepo *client.NotaryRepository) error {

	cl, err := notaryRepo.GetChangelist()
	if err != nil {
		return err
	}
	if err = cl.Clear(""); err != nil {
		return err
	}
	return nil
}
