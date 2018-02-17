package trust

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/notary/client"
)

func newViewCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view IMAGE[:TAG]",
		Short: "Display detailed information about keys and signatures",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return viewTrustInfo(dockerCli, args[0])
		},
	}
	return cmd
}

func viewTrustInfo(cli command.Cli, remote string) error {
	signatureRows, adminRolesWithSigs, delegationRoles, err := lookupTrustInfo(cli, remote)
	if err != nil {
		return err
	}

	if len(signatureRows) > 0 {
		if err := printSignatures(cli.Out(), signatureRows); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(cli.Out(), "\nNo signatures for %s\n\n", remote)
	}
	signerRoleToKeyIDs := getDelegationRoleToKeyMap(delegationRoles)

	// If we do not have additional signers, do not display
	if len(signerRoleToKeyIDs) > 0 {
		fmt.Fprintf(cli.Out(), "\nList of signers and their keys for %s:\n\n", strings.Split(remote, ":")[0])
		if err := printSignerInfo(cli.Out(), signerRoleToKeyIDs); err != nil {
			return err
		}
	}

	// This will always have the root and targets information
	fmt.Fprintf(cli.Out(), "\nAdministrative keys for %s:\n", strings.Split(remote, ":")[0])
	printSortedAdminKeys(cli.Out(), adminRolesWithSigs)
	return nil
}

func printSortedAdminKeys(out io.Writer, adminRoles []client.RoleWithSignatures) {
	sort.Slice(adminRoles, func(i, j int) bool { return adminRoles[i].Name > adminRoles[j].Name })
	for _, adminRole := range adminRoles {
		fmt.Fprintf(out, "%s", formatAdminRole(adminRole))
	}
}

// pretty print with ordered rows
func printSignatures(out io.Writer, signatureRows trustTagRowList) error {
	trustTagCtx := formatter.Context{
		Output: out,
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
			Name:    sigRow.SignedTag,
			Digest:  sigRow.Digest,
			Signers: formattedSigners,
		})
	}
	return formatter.TrustTagWrite(trustTagCtx, formattedTags)
}

func printSignerInfo(out io.Writer, roleToKeyIDs map[string][]string) error {
	signerInfoCtx := formatter.Context{
		Output: out,
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
