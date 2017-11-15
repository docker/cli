package trust

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/notary/tuf/data"
)

func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect IMAGE[:TAG]",
		Short: "Return low-level information about keys and signatures",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return inspectTrustInfo(dockerCli, args[0])
		},
	}
	return cmd
}

func inspectTrustInfo(cli command.Cli, remote string) error {
	signatureRows, adminRolesWithSigs, delegationRoles, err := lookupTrustInfo(cli, remote)
	if err != nil {
		return err
	}
	// process the signatures to include repo admin if signed by the base targets role
	for idx, sig := range signatureRows {
		if len(sig.Signers) == 0 {
			signatureRows[idx].Signers = append(sig.Signers, releasedRoleName)
		}
	}

	signerList, adminList := []trustSigner{}, []trustSigner{}

	signerRoleToKeyIDs := getDelegationRoleToKeyMap(delegationRoles)

	for signerName, signerKeys := range signerRoleToKeyIDs {
		signerList = append(signerList, trustSigner{signerName, signerKeys})
	}
	sort.Slice(signerList, func(i, j int) bool { return signerList[i].Name > signerList[j].Name })

	for _, adminRole := range adminRolesWithSigs {
		switch adminRole.Name {
		case data.CanonicalRootRole:
			adminList = append(adminList, trustSigner{"Root", adminRole.KeyIDs})
		case data.CanonicalTargetsRole:
			adminList = append(adminList, trustSigner{"Repository", adminRole.KeyIDs})
		}
	}
	sort.Slice(adminList, func(i, j int) bool { return adminList[i].Name > adminList[j].Name })

	trustRepoInfo := &trustRepo{
		SignedTags:        signatureRows,
		Signers:           signerList,
		AdminstrativeKeys: adminList,
	}
	trustInspectJSON, err := json.Marshal(trustRepoInfo)
	if err != nil {
		return errors.Wrap(err, "error while serializing trusted repository info")
	}
	fmt.Fprintf(cli.Out(), string(trustInspectJSON))
	return nil
}

// trustRepo represents consumable information about a trusted repository
type trustRepo struct {
	SignedTags        trustTagRowList `json:",omitempty"`
	Signers           []trustSigner   `json:",omitempty"`
	AdminstrativeKeys []trustSigner   `json:",omitempty"`
}

// trustSigner represents a trusted signer in a trusted repository
// a signer is defined by a name and list of key IDs
type trustSigner struct {
	Name string   `json:",omitempty"`
	Keys []string `json:",omitempty"`
}
