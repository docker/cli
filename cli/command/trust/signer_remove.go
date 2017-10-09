package trust

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/docker/api/types"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/notary/client"
	"github.com/docker/notary/tuf/data"
	"github.com/spf13/cobra"
)

type signerRemoveOptions struct {
	forceYes bool
}

func newSignerRemoveCommand(dockerCli command.Cli) *cobra.Command {
	options := signerRemoveOptions{}
	cmd := &cobra.Command{
		Use:   "remove [OPTIONS] NAME IMAGE [IMAGE...]",
		Short: "Remove a signer",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeSigner(dockerCli, args[0], args[1:], &options)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&options.forceYes, "yes", "y", false, "Answer yes to removing most recent signer (no confirmation)")
	return cmd
}

func removeSigner(cli command.Cli, signer string, images []string, options *signerRemoveOptions) error {
	var errImages []string
	for _, image := range images {
		if err := removeSingleSigner(cli, image, signer, options.forceYes); err != nil {
			fmt.Fprintln(cli.Out(), err.Error())
			errImages = append(errImages, image)
		}
	}
	if len(errImages) > 0 {
		return fmt.Errorf("Error removing signer from: %s", strings.Join(errImages, ", "))
	}
	return nil
}

func isLastSignerForReleases(roleWithSig data.Role, allRoles []client.RoleWithSignatures) (bool, error) {
	var releasesRoleWithSigs client.RoleWithSignatures
	for _, role := range allRoles {
		if role.Name == releasesRoleTUFName {
			releasesRoleWithSigs = role
			break
		}
	}
	counter := len(releasesRoleWithSigs.Signatures)
	if counter == 0 {
		return false, fmt.Errorf("All signed tags are currently revoked, use docker trust sign to fix")
	}
	for _, signature := range releasesRoleWithSigs.Signatures {
		for _, key := range roleWithSig.KeyIDs {
			if signature.KeyID == key {
				counter--
			}
		}
	}
	return counter < releasesRoleWithSigs.Threshold, nil
}

func removeSingleSigner(cli command.Cli, image, signerName string, forceYes bool) error {
	fmt.Fprintf(cli.Out(), "\nRemoving signer \"%s\" from %s...\n", signerName, image)

	ctx := context.Background()
	authResolver := func(ctx context.Context, index *registrytypes.IndexInfo) types.AuthConfig {
		return command.ResolveAuthConfig(ctx, cli, index)
	}
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, authResolver, image)
	if err != nil {
		return err
	}

	signerDelegation := data.RoleName("targets/" + signerName)
	if signerDelegation == releasesRoleTUFName {
		return fmt.Errorf("releases is a reserved keyword and cannot be removed")
	}
	notaryRepo, err := cli.NotaryClient(*imgRefAndAuth, trust.ActionsPushAndPull)
	if err != nil {
		return trust.NotaryError(imgRefAndAuth.Reference().Name(), err)
	}
	delegationRoles, err := notaryRepo.GetDelegationRoles()
	if err != nil {
		return fmt.Errorf("Error retrieving signers for %s", image)
	}
	var role data.Role
	for _, delRole := range delegationRoles {
		if delRole.Name == signerDelegation {
			role = delRole
			break
		}
	}
	if role.Name == "" {
		return fmt.Errorf("No signer %s for image %s", signerName, image)
	}
	allRoles, err := notaryRepo.ListRoles()
	if err != nil {
		return err
	}
	if ok, err := isLastSignerForReleases(role, allRoles); ok && !forceYes {
		removeSigner := command.PromptForConfirmation(os.Stdin, cli.Out(), fmt.Sprintf("The signer \"%s\" signed the last released version of %s. "+
			"Removing this signer will make %s unpullable. "+
			"Are you sure you want to continue?",
			signerName, image, image,
		))

		if !removeSigner {
			fmt.Fprintf(cli.Out(), "\nAborting action.\n")
			return nil
		}
	} else if err != nil {
		fmt.Fprintln(cli.Out(), err.Error())
	}
	if err = notaryRepo.RemoveDelegationKeys(releasesRoleTUFName, role.KeyIDs); err != nil {
		return err
	}
	if err = notaryRepo.RemoveDelegationRole(signerDelegation); err != nil {
		return err
	}
	if err = notaryRepo.Publish(); err != nil {
		return err
	}
	fmt.Fprintf(cli.Out(), "Successfully removed %s from %s\n", signerName, image)
	return nil
}
