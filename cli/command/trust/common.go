package trust

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/trust"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
)

// trustTagKey represents a unique signed tag and hex-encoded hash pair
type trustTagKey struct {
	SignedTag string
	Digest    string
}

// trustTagRow encodes all human-consumable information for a signed tag, including signers
type trustTagRow struct {
	trustTagKey
	Signers []string
}

type trustTagRowList []trustTagRow

func (tagComparator trustTagRowList) Len() int {
	return len(tagComparator)
}

func (tagComparator trustTagRowList) Less(i, j int) bool {
	return tagComparator[i].SignedTag < tagComparator[j].SignedTag
}

func (tagComparator trustTagRowList) Swap(i, j int) {
	tagComparator[i], tagComparator[j] = tagComparator[j], tagComparator[i]
}

// trustRepo represents consumable information about a trusted repository
type trustRepo struct {
	Name              string
	SignedTags        trustTagRowList
	Signers           []trustSigner
	AdminstrativeKeys []trustSigner
}

// trustSigner represents a trusted signer in a trusted repository
// a signer is defined by a name and list of trustKeys
type trustSigner struct {
	Name string     `json:",omitempty"`
	Keys []trustKey `json:",omitempty"`
}

// trustKey contains information about trusted keys
type trustKey struct {
	ID string `json:",omitempty"`
}

// lookupTrustInfo returns processed signature and role information about a notary repository.
// This information is to be pretty printed or serialized into a machine-readable format.
func lookupTrustInfo(cli command.Cli, remote string) (trustTagRowList, []client.RoleWithSignatures, []data.Role, error) {
	ctx := context.Background()
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, nil, image.AuthResolver(cli), remote)
	if err != nil {
		return trustTagRowList{}, []client.RoleWithSignatures{}, []data.Role{}, err
	}
	tag := imgRefAndAuth.Tag()
	notaryRepo, err := cli.NotaryClient(imgRefAndAuth, trust.ActionsPullOnly)
	if err != nil {
		return trustTagRowList{}, []client.RoleWithSignatures{}, []data.Role{}, trust.NotaryError(imgRefAndAuth.Reference().Name(), err)
	}

	if err = clearChangeList(notaryRepo); err != nil {
		return trustTagRowList{}, []client.RoleWithSignatures{}, []data.Role{}, err
	}
	defer clearChangeList(notaryRepo)

	// Retrieve all released signatures, match them, and pretty print them
	allSignedTargets, err := notaryRepo.GetAllTargetMetadataByName(tag)
	if err != nil {
		logrus.Debug(trust.NotaryError(remote, err))
		// print an empty table if we don't have signed targets, but have an initialized notary repo
		if _, ok := err.(client.ErrNoSuchTarget); !ok {
			return trustTagRowList{}, []client.RoleWithSignatures{}, []data.Role{}, fmt.Errorf("No signatures or cannot access %s", remote)
		}
	}
	signatureRows := matchReleasedSignatures(allSignedTargets)

	// get the administrative roles
	adminRolesWithSigs, err := notaryRepo.ListRoles()
	if err != nil {
		return trustTagRowList{}, []client.RoleWithSignatures{}, []data.Role{}, fmt.Errorf("No signers for %s", remote)
	}

	// get delegation roles with the canonical key IDs
	delegationRoles, err := notaryRepo.GetDelegationRoles()
	if err != nil {
		logrus.Debugf("no delegation roles found, or error fetching them for %s: %v", remote, err)
	}

	return signatureRows, adminRolesWithSigs, delegationRoles, nil
}

func formatAdminRole(roleWithSigs client.RoleWithSignatures) string {
	adminKeyList := roleWithSigs.KeyIDs
	sort.Strings(adminKeyList)

	var role string
	switch roleWithSigs.Name {
	case data.CanonicalTargetsRole:
		role = "Repository Key"
	case data.CanonicalRootRole:
		role = "Root Key"
	default:
		return ""
	}
	return fmt.Sprintf("%s:\t%s\n", role, strings.Join(adminKeyList, ", "))
}

func getDelegationRoleToKeyMap(rawDelegationRoles []data.Role) map[string][]string {
	signerRoleToKeyIDs := make(map[string][]string)
	for _, delRole := range rawDelegationRoles {
		switch delRole.Name {
		case trust.ReleasesRole, data.CanonicalRootRole, data.CanonicalSnapshotRole, data.CanonicalTargetsRole, data.CanonicalTimestampRole:
			continue
		default:
			signerRoleToKeyIDs[notaryRoleToSigner(delRole.Name)] = delRole.KeyIDs
		}
	}
	return signerRoleToKeyIDs
}

// aggregate all signers for a "released" hash+tagname pair. To be "released," the tag must have been
// signed into the "targets" or "targets/releases" role. Output is sorted by tag name
func matchReleasedSignatures(allTargets []client.TargetSignedStruct) trustTagRowList {
	signatureRows := trustTagRowList{}
	// do a first pass to get filter on tags signed into "targets" or "targets/releases"
	releasedTargetRows := map[trustTagKey][]string{}
	for _, tgt := range allTargets {
		if isReleasedTarget(tgt.Role.Name) {
			releasedKey := trustTagKey{tgt.Target.Name, hex.EncodeToString(tgt.Target.Hashes[notary.SHA256])}
			releasedTargetRows[releasedKey] = []string{}
		}
	}

	// now fill out all signers on released keys
	for _, tgt := range allTargets {
		targetKey := trustTagKey{tgt.Target.Name, hex.EncodeToString(tgt.Target.Hashes[notary.SHA256])}
		// only considered released targets
		if _, ok := releasedTargetRows[targetKey]; ok && !isReleasedTarget(tgt.Role.Name) {
			releasedTargetRows[targetKey] = append(releasedTargetRows[targetKey], notaryRoleToSigner(tgt.Role.Name))
		}
	}

	// compile the final output as a sorted slice
	for targetKey, signers := range releasedTargetRows {
		signatureRows = append(signatureRows, trustTagRow{targetKey, signers})
	}
	sort.Sort(signatureRows)
	return signatureRows
}
