package trust

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/notary"
	"github.com/docker/notary/client"
	"github.com/docker/notary/tuf/data"
	"github.com/spf13/cobra"
)

// trustTagKey represents a unique signed tag and hex-encoded hash pair
type trustTagKey struct {
	TagName string
	HashHex string
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
	return tagComparator[i].TagName < tagComparator[j].TagName
}

func (tagComparator trustTagRowList) Swap(i, j int) {
	tagComparator[i], tagComparator[j] = tagComparator[j], tagComparator[i]
}

func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] IMAGE[:TAG]",
		Short: "Display detailed information about keys and signatures",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return lookupTrustInfo(dockerCli, args[0])
		},
	}
	return cmd
}

func lookupTrustInfo(cli command.Cli, remote string) error {
	_, ref, repoInfo, authConfig, err := getImageReferencesAndAuth(cli, remote)
	if err != nil {
		return err
	}
	notaryRepo, err := trust.GetNotaryRepository(cli, repoInfo, *authConfig, "pull")
	if err != nil {
		return trust.NotaryError(ref.Name(), err)
	}

	if err = clearChangeList(notaryRepo); err != nil {
		return err
	}
	defer clearChangeList(notaryRepo)
	tag, err := getTag(ref)
	if err != nil {
		return err
	}
	// Retrieve all released signatures, match them, and pretty print them
	allSignedTargets, err := notaryRepo.GetAllTargetMetadataByName(tag)
	if err != nil {
		logrus.Debug(trust.NotaryError(ref.Name(), err))
		// print an empty table if we don't have signed targets, but have an initialized notary repo
		if _, ok := err.(client.ErrNoSuchTarget); !ok {
			return fmt.Errorf("No signatures or cannot access %s", remote)
		}
	}
	signatureRows := matchReleasedSignatures(allSignedTargets)
	if len(signatureRows) > 0 {
		if err := printSignatures(cli, signatureRows); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(cli.Out(), "\nNo signatures for %s\n\n", remote)
	}

	// get the administrative roles
	roleWithSigs, err := notaryRepo.ListRoles()
	if err != nil {
		return fmt.Errorf("No signers for %s", remote)
	}
	adminRoleToKeyIDs := getAdministrativeRolesToKeyMap(roleWithSigs)

	// get delegation roles with the canonical key IDs
	delegationRoles, err := notaryRepo.GetDelegationRoles()
	if err != nil {
		logrus.Debugf("no delegation roles found, or error fetching them for %s: %v", remote, err)
	}
	signerRoleToKeyIDs := getDelegationRoleToKeyMap(delegationRoles)

	// If we do not have additional signers, do not display
	if len(signerRoleToKeyIDs) > 0 {
		fmt.Fprintf(cli.Out(), "\nList of signers and their KeyIDs:\n\n")
		printSignerInfo(cli, signerRoleToKeyIDs)
	}

	// This will always have the root and targets information
	fmt.Fprintf(cli.Out(), "\nAdministrative keys for %s:\n", strings.Split(remote, ":")[0])
	printSortedAdminKeys(adminRoleToKeyIDs, cli)

	return nil
}

func printSortedAdminKeys(adminRoleToKeyIDs map[string]string, cli command.Cli) {
	keyNames := []string{}
	for name := range adminRoleToKeyIDs {
		keyNames = append(keyNames, name)
	}

	sort.Strings(keyNames)

	for _, keyName := range keyNames {
		fmt.Fprintf(cli.Out(), "%s:\t%s\n", keyName, adminRoleToKeyIDs[keyName])
	}
}

func getAdministrativeRolesToKeyMap(roleWithSigs []client.RoleWithSignatures) map[string]string {
	adminRoleToKeyIDs := make(map[string]string)
	for _, roleWithSig := range roleWithSigs {
		sort.Strings(roleWithSig.KeyIDs)
		switch roleWithSig.Name {
		case data.CanonicalTargetsRole:
			adminRoleToKeyIDs["Repository Key"] = strings.Join(roleWithSig.KeyIDs, ", ")
		case data.CanonicalRootRole:
			adminRoleToKeyIDs["Root Key"] = strings.Join(roleWithSig.KeyIDs, ", ")
		default:
			continue
		}
	}
	return adminRoleToKeyIDs
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

// pretty print with ordered rows
func printSignatures(dockerCli command.Cli, signatureRows trustTagRowList) error {
	trustTagCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewTrustTagFormat(),
	}
	// convert the formatted type before printing
	formattedTags := []formatter.SignedTagInfo{}
	for _, sigRow := range signatureRows {
		formattedSigners := sigRow.Signers
		if len(formattedSigners) == 0 {
			formattedSigners = append(formattedSigners, fmt.Sprintf("(%s)", releasedRoleName))
		}
		formattedTags = append(formattedTags, formatter.SignedTagInfo{
			Name:    sigRow.TagName,
			Digest:  sigRow.HashHex,
			Signers: formattedSigners,
		})
	}
	return formatter.TrustTagWrite(trustTagCtx, formattedTags)
}

func printSignerInfo(dockerCli command.Cli, roleToKeyIDs map[string][]string) error {
	signerInfoCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewSignerInfoFormat(),
		Trunc:  true,
	}
	formattedSignerInfo := formatter.SignerInfoList{}
	for name, keyIDs := range roleToKeyIDs {
		formattedSignerInfo = append(formattedSignerInfo, formatter.SignerInfo{
			Name: name,
			Keys: keyIDs,
		})
	}
	sort.Sort(formattedSignerInfo)
	return formatter.SignerInfoWrite(signerInfoCtx, formattedSignerInfo)
}
