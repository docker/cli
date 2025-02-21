package trust // import "docker.com/cli/v28/cli/command/trust"

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/docker/cli/v28/cli/command"
	"github.com/docker/cli/v28/cli/command/formatter"
	"github.com/fvbommel/sortorder"
	"github.com/theupdateframework/notary/client"
)

func prettyPrintTrustInfo(ctx context.Context, dockerCLI command.Cli, remote string) error {
	signatureRows, adminRolesWithSigs, delegationRoles, err := lookupTrustInfo(ctx, dockerCLI, remote)
	if err != nil {
		return err
	}

	if len(signatureRows) > 0 {
		_, _ = fmt.Fprintf(dockerCLI.Out(), "\nSignatures for %s\n\n", remote)

		if err := printSignatures(dockerCLI.Out(), signatureRows); err != nil {
			return err
		}
	} else {
		_, _ = fmt.Fprintf(dockerCLI.Out(), "\nNo signatures for %s\n\n", remote)
	}
	signerRoleToKeyIDs := getDelegationRoleToKeyMap(delegationRoles)

	// If we do not have additional signers, do not display
	if len(signerRoleToKeyIDs) > 0 {
		_, _ = fmt.Fprintf(dockerCLI.Out(), "\nList of signers and their keys for %s\n\n", remote)
		if err := printSignerInfo(dockerCLI.Out(), signerRoleToKeyIDs); err != nil {
			return err
		}
	}

	// This will always have the root and targets information
	_, _ = fmt.Fprintf(dockerCLI.Out(), "\nAdministrative keys for %s\n\n", remote)
	printSortedAdminKeys(dockerCLI.Out(), adminRolesWithSigs)
	return nil
}

func printSortedAdminKeys(out io.Writer, adminRoles []client.RoleWithSignatures) {
	sort.Slice(adminRoles, func(i, j int) bool { return adminRoles[i].Name > adminRoles[j].Name })
	for _, adminRole := range adminRoles {
		if formattedAdminRole := formatAdminRole(adminRole); formattedAdminRole != "" {
			_, _ = fmt.Fprintf(out, "  %s", formattedAdminRole)
		}
	}
}

// pretty print with ordered rows
func printSignatures(out io.Writer, signatureRows []trustTagRow) error {
	trustTagCtx := formatter.Context{
		Output: out,
		Format: NewTrustTagFormat(),
	}
	// convert the formatted type before printing
	formattedTags := []SignedTagInfo{}
	for _, sigRow := range signatureRows {
		formattedSigners := sigRow.Signers
		if len(formattedSigners) == 0 {
			formattedSigners = append(formattedSigners, fmt.Sprintf("(%s)", releasedRoleName))
		}
		formattedTags = append(formattedTags, SignedTagInfo{
			Name:    sigRow.SignedTag,
			Digest:  sigRow.Digest,
			Signers: formattedSigners,
		})
	}
	return TagWrite(trustTagCtx, formattedTags)
}

func printSignerInfo(out io.Writer, roleToKeyIDs map[string][]string) error {
	signerInfoCtx := formatter.Context{
		Output: out,
		Format: NewSignerInfoFormat(),
		Trunc:  true,
	}
	formattedSignerInfo := []SignerInfo{}
	for name, keyIDs := range roleToKeyIDs {
		formattedSignerInfo = append(formattedSignerInfo, SignerInfo{
			Name: name,
			Keys: keyIDs,
		})
	}
	sort.Slice(formattedSignerInfo, func(i, j int) bool {
		return sortorder.NaturalLess(formattedSignerInfo[i].Name, formattedSignerInfo[j].Name)
	})
	return SignerInfoWrite(signerInfoCtx, formattedSignerInfo)
}
