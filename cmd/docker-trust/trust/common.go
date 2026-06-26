package trust

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cmd/docker-trust/internal/trust"
	"github.com/fvbommel/sortorder"
	registrytypes "github.com/moby/moby/api/types/registry"
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

// trustRepo represents consumable information about a trusted repository
type trustRepo struct {
	Name               string
	SignedTags         []trustTagRow
	Signers            []trustSigner
	AdministrativeKeys []trustSigner
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

// notaryClientProvider is used in tests to provide a dummy notary client.
type notaryClientProvider interface {
	NotaryClient() (client.Repository, error)
}

// newNotaryClient provides a Notary Repository to interact with signed metadata for an image.
func newNotaryClient(cli command.Streams, imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	if ncp, ok := cli.(notaryClientProvider); ok {
		// notaryClientProvider is used in tests to provide a dummy notary client.
		return ncp.NotaryClient()
	}
	return trust.GetNotaryRepository(cli.In(), cli.Out(), command.UserAgent(), imgRefAndAuth.RepoInfo(), imgRefAndAuth.AuthConfig(), actions...)
}

// lookupTrustInfo returns processed signature and role information about a notary repository.
// This information is to be pretty printed or serialized into a machine-readable format.
func lookupTrustInfo(ctx context.Context, cli command.Cli, remote string) ([]trustTagRow, []client.RoleWithSignatures, []data.Role, error) {
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, authResolver(cli), remote)
	if err != nil {
		return []trustTagRow{}, []client.RoleWithSignatures{}, []data.Role{}, err
	}
	tag := imgRefAndAuth.Tag()
	notaryRepo, err := newNotaryClient(cli, imgRefAndAuth, trust.ActionsPullOnly)
	if err != nil {
		return []trustTagRow{}, []client.RoleWithSignatures{}, []data.Role{}, trust.NotaryError(imgRefAndAuth.Reference().Name(), err)
	}

	if err = clearChangeList(notaryRepo); err != nil {
		return []trustTagRow{}, []client.RoleWithSignatures{}, []data.Role{}, err
	}
	defer clearChangeList(notaryRepo)

	// Retrieve all released signatures, match them, and pretty print them
	allSignedTargets, err := notaryRepo.GetAllTargetMetadataByName(tag)
	if err != nil {
		logrus.Debug(trust.NotaryError(remote, err))
		// print an empty table if we don't have signed targets, but have an initialized notary repo
		if _, ok := err.(client.ErrNoSuchTarget); !ok {
			return []trustTagRow{}, []client.RoleWithSignatures{}, []data.Role{}, fmt.Errorf("no signatures or cannot access %s", remote)
		}
	}
	signatureRows := matchReleasedSignatures(allSignedTargets)

	// get the administrative roles
	adminRolesWithSigs, err := notaryRepo.ListRoles()
	if err != nil {
		return []trustTagRow{}, []client.RoleWithSignatures{}, []data.Role{}, fmt.Errorf("no signers for %s", remote)
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
func matchReleasedSignatures(allTargets []client.TargetSignedStruct) []trustTagRow {
	signatureRows := []trustTagRow{}
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
	sort.Slice(signatureRows, func(i, j int) bool {
		return sortorder.NaturalLess(signatureRows[i].SignedTag, signatureRows[j].SignedTag)
	})
	return signatureRows
}

// authResolver returns an auth resolver function from a [config.Provider].
func authResolver(dockerCLI config.Provider) func(ctx context.Context, index *registrytypes.IndexInfo) registrytypes.AuthConfig {
	return func(ctx context.Context, index *registrytypes.IndexInfo) registrytypes.AuthConfig {
		return resolveAuthConfig(dockerCLI.ConfigFile(), index)
	}
}

// authConfigKey is the key used to store credentials for Docker Hub. It is
// a copy of [registry.IndexServer].
//
// [registry.IndexServer]: https://pkg.go.dev/github.com/docker/docker@v28.3.3+incompatible/registry#IndexServer
const authConfigKey = "https://index.docker.io/v1/"

// resolveAuthConfig returns auth-config for the given registry from the
// credential-store. It returns an empty AuthConfig if no credentials were
// found.
//
// It is similar to [registry.ResolveAuthConfig], but uses the credentials-
// store, instead of looking up credentials from a map.
//
// [registry.ResolveAuthConfig]: https://pkg.go.dev/github.com/docker/docker@v28.3.3+incompatible/registry#ResolveAuthConfig
func resolveAuthConfig(cfg *configfile.ConfigFile, index *registrytypes.IndexInfo) registrytypes.AuthConfig {
	configKey := index.Name
	if index.Official {
		configKey = authConfigKey
	}

	a, _ := cfg.GetAuthConfig(configKey)
	return registrytypes.AuthConfig{
		Username:      a.Username,
		Password:      a.Password,
		ServerAddress: a.ServerAddress,

		// TODO(thaJeztah): Are these expected to be included?
		Auth:          a.Auth,
		IdentityToken: a.IdentityToken,
		RegistryToken: a.RegistryToken,
	}
}
