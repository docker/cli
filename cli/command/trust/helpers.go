package trust

import (
	"strings"

	"github.com/docker/cli/cli/trust"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
)

const releasedRoleName = "Repo Admin"
const releasesRoleTUFName = "targets/releases"

// isReleasedTarget checks if a role name is "released":
// either targets/releases or targets TUF roles
func isReleasedTarget(role data.RoleName) bool {
	return role == data.CanonicalTargetsRole || role == trust.ReleasesRole
}

// notaryRoleToSigner converts TUF role name to a human-understandable signer name
func notaryRoleToSigner(tufRole data.RoleName) string {
	//  don't show a signer for "targets" or "targets/releases"
	if isReleasedTarget(data.RoleName(tufRole.String())) {
		return releasedRoleName
	}
	return strings.TrimPrefix(tufRole.String(), "targets/")
}

// clearChangelist clears the notary staging changelist.
func clearChangeList(notaryRepo client.Repository) error {
	cl, err := notaryRepo.GetChangelist()
	if err != nil {
		return err
	}
	return cl.Clear("")
}

// getOrGenerateRootKeyAndInitRepo initializes the notary repository
// with a remotely managed snapshot key. The initialization will use
// an existing root key if one is found, else a new one will be generated.
func getOrGenerateRootKeyAndInitRepo(notaryRepo client.Repository) error {
	rootKey, err := getOrGenerateNotaryKey(notaryRepo, data.CanonicalRootRole)
	if err != nil {
		return err
	}
	return notaryRepo.Initialize([]string{rootKey.ID()}, data.CanonicalSnapshotRole)
}
