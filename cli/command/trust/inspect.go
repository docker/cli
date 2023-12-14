// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package trust

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/notary/tuf/data"
)

type inspectOptions struct {
	remotes []string
	// FIXME(n4ss): this is consistent with `docker service inspect` but we should provide
	// a `--format` flag too. (format and pretty-print should be exclusive)
	prettyPrint bool
}

func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	options := inspectOptions{}
	cmd := &cobra.Command{
		Use:   "inspect IMAGE[:TAG] [IMAGE[:TAG]...]",
		Short: "Return low-level information about keys and signatures",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.remotes = args

			return runInspect(dockerCli, options)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&options.prettyPrint, "pretty", false, "Print the information in a human friendly format")

	return cmd
}

func runInspect(dockerCli command.Cli, opts inspectOptions) error {
	if opts.prettyPrint {
		var err error

		for index, remote := range opts.remotes {
			if err = prettyPrintTrustInfo(dockerCli, remote); err != nil {
				return err
			}

			// Additional separator between the inspection output of each image
			if index < len(opts.remotes)-1 {
				fmt.Fprint(dockerCli.Out(), "\n\n")
			}
		}

		return err
	}

	getRefFunc := func(ref string) (interface{}, []byte, error) {
		i, err := getRepoTrustInfo(dockerCli, ref)
		return nil, i, err
	}
	return inspect.Inspect(dockerCli.Out(), opts.remotes, "", getRefFunc)
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
		Name:               remote,
		SignedTags:         signatureRows,
		Signers:            signerList,
		AdministrativeKeys: adminList,
	})
}
