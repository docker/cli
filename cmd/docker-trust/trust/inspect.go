// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package trust

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"

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

func newInspectCommand(dockerCLI command.Cli) *cobra.Command {
	options := inspectOptions{}
	cmd := &cobra.Command{
		Use:   "inspect IMAGE[:TAG] [IMAGE[:TAG]...]",
		Short: "Return low-level information about keys and signatures",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.remotes = args

			return runInspect(cmd.Context(), dockerCLI, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVar(&options.prettyPrint, "pretty", false, "Print the information in a human friendly format")

	return cmd
}

func runInspect(ctx context.Context, dockerCLI command.Cli, opts inspectOptions) error {
	if opts.prettyPrint {
		var err error

		for index, remote := range opts.remotes {
			if err = prettyPrintTrustInfo(ctx, dockerCLI, remote); err != nil {
				return err
			}

			// Additional separator between the inspection output of each image
			if index < len(opts.remotes)-1 {
				_, _ = fmt.Fprint(dockerCLI.Out(), "\n\n")
			}
		}

		return err
	}

	getRefFunc := func(ref string) (any, []byte, error) {
		i, err := getRepoTrustInfo(ctx, dockerCLI, ref)
		return nil, i, err
	}
	return inspect.Inspect(dockerCLI.Out(), opts.remotes, "", getRefFunc)
}

func getRepoTrustInfo(ctx context.Context, dockerCLI command.Cli, remote string) ([]byte, error) {
	signatureRows, adminRolesWithSigs, delegationRoles, err := lookupTrustInfo(ctx, dockerCLI, remote)
	if err != nil {
		return []byte{}, err
	}
	// process the signatures to include repo admin if signed by the base targets role
	for idx, sig := range signatureRows {
		if len(sig.Signers) == 0 {
			signatureRows[idx].Signers = []string{releasedRoleName}
		}
	}

	signerList := []trustSigner{}
	for signerName, signerKeys := range getDelegationRoleToKeyMap(delegationRoles) {
		signerKeyList := make([]trustKey, 0, len(signerKeys))
		for _, keyID := range signerKeys {
			signerKeyList = append(signerKeyList, trustKey{ID: keyID})
		}
		signerList = append(signerList, trustSigner{signerName, signerKeyList})
	}
	// Sort by name in descending order.
	slices.SortFunc(signerList, func(a, b trustSigner) int { return cmp.Compare(b.Name, a.Name) })

	var adminList []trustSigner
	for _, adminRole := range adminRolesWithSigs {
		var name string
		switch adminRole.Name {
		case data.CanonicalRootRole:
			name = "Root"
		case data.CanonicalTargetsRole:
			name = "Repository"
		default:
			continue
		}

		keys := make([]trustKey, 0, len(adminRole.KeyIDs))
		for _, keyID := range adminRole.KeyIDs {
			keys = append(keys, trustKey{ID: keyID})
		}
		adminList = append(adminList, trustSigner{name, keys})
	}

	// Sort by name in descending order.
	slices.SortFunc(adminList, func(a, b trustSigner) int { return cmp.Compare(b.Name, a.Name) })

	return json.Marshal(trustRepo{
		Name:               remote,
		SignedTags:         signatureRows,
		Signers:            signerList,
		AdministrativeKeys: adminList,
	})
}
