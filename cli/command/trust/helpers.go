package trust

import (
	"strings"

	"github.com/docker/cli/cli/trust"
	"github.com/docker/notary/client"
	"github.com/docker/notary/tuf/data"
)

const releasedRoleName = "Repo Admin"
const releasesRoleTUFName = "targets/releases"

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

func getOrGenerateRootKeyAndInitRepo(notaryRepo client.Repository) error {
	rootKey, err := getOrGenerateNotaryKey(notaryRepo, data.CanonicalRootRole)
	if err != nil {
		return err
	}
	// Initialize the notary repository with a remotely managed snapshot
	// key
	return notaryRepo.Initialize([]string{rootKey.ID()}, data.CanonicalSnapshotRole)
}
