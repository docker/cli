package trust

import (
	"encoding/json"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/notary/tuf/data"
)

func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect IMAGE[:TAG] [IMAGE[:TAG]...]",
		Short: "Return low-level information about keys and signatures",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(dockerCli, args)
		},
	}
	return cmd
}

func runInspect(dockerCli command.Cli, remotes []string) error {
	getRefFunc := func(ref string) (interface{}, []byte, error) {
		i, err := getRepoTrustInfo(dockerCli, ref)
		return nil, i, err
	}
	return inspect.Inspect(dockerCli.Out(), remotes, "", getRefFunc)
}

func getRepoTrustInfo(cli command.Cli, remote string) ([]byte, error) {
	signatureRows, adminRolesWithSigs, delegationRoles, err := lookupTrustInfo(cli, remote)
	if err != nil {
		return []byte{}, err
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
		signerKeyList := []trustKey{}
		for _, keyID := range signerKeys {
			signerKeyList = append(signerKeyList, trustKey{ID: keyID})
		}
		signerList = append(signerList, trustSigner{signerName, signerKeyList})
	}
	sort.Slice(signerList, func(i, j int) bool { return signerList[i].Name > signerList[j].Name })

	for _, adminRole := range adminRolesWithSigs {
		switch adminRole.Name {
		case data.CanonicalRootRole:
			rootKeys := []trustKey{}
			for _, keyID := range adminRole.KeyIDs {
				rootKeys = append(rootKeys, trustKey{ID: keyID})
			}
			adminList = append(adminList, trustSigner{"Root", rootKeys})
		case data.CanonicalTargetsRole:
			targetKeys := []trustKey{}
			for _, keyID := range adminRole.KeyIDs {
				targetKeys = append(targetKeys, trustKey{ID: keyID})
			}
			adminList = append(adminList, trustSigner{"Repository", targetKeys})
		}
	}
	sort.Slice(adminList, func(i, j int) bool { return adminList[i].Name > adminList[j].Name })

	return json.Marshal(trustRepo{
		Name:              remote,
		SignedTags:        signatureRows,
		Signers:           signerList,
		AdminstrativeKeys: adminList,
	})
}
