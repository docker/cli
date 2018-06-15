package trust

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/trust"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
)

type signerRemoveOptions struct {
	signer   string
	repos    []string
	forceYes bool
}

func newSignerRemoveCommand(dockerCli command.Cli) *cobra.Command {
	options := signerRemoveOptions{}
	cmd := &cobra.Command{
		Use:   "remove [OPTIONS] NAME REPOSITORY [REPOSITORY...]",
		Short: "Remove a signer",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.signer = args[0]
			options.repos = args[1:]
			return removeSigner(dockerCli, options)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&options.forceYes, "force", "f", false, "Do not prompt for confirmation before removing the most recent signer")
	return cmd
}

func removeSigner(cli command.Cli, options signerRemoveOptions) error {
	var errRepos []string
	for _, repo := range options.repos {
		fmt.Fprintf(cli.Out(), "Removing signer \"%s\" from %s...\n", options.signer, repo)
		if _, err := removeSingleSigner(cli, repo, options.signer, options.forceYes); err != nil {
			fmt.Fprintln(cli.Err(), err.Error()+"\n")
			errRepos = append(errRepos, repo)
		}
	}
	if len(errRepos) > 0 {
		return fmt.Errorf("Error removing signer from: %s", strings.Join(errRepos, ", "))
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

// removeSingleSigner attempts to remove a single signer and returns whether signer removal happened.
// The signer not being removed doesn't necessarily raise an error e.g. user choosing "No" when prompted for confirmation.
func removeSingleSigner(cli command.Cli, repoName, signerName string, forceYes bool) (bool, error) {
	ctx := context.Background()
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, nil, image.AuthResolver(cli), repoName)
	if err != nil {
		return false, err
	}

	signerDelegation := data.RoleName("targets/" + signerName)
	if signerDelegation == releasesRoleTUFName {
		return false, fmt.Errorf("releases is a reserved keyword and cannot be removed")
	}
	notaryRepo, err := cli.NotaryClient(imgRefAndAuth, trust.ActionsPushAndPull)
	if err != nil {
		return false, trust.NotaryError(imgRefAndAuth.Reference().Name(), err)
	}
	delegationRoles, err := notaryRepo.GetDelegationRoles()
	if err != nil {
		return false, errors.Wrapf(err, "error retrieving signers for %s", repoName)
	}
	var role data.Role
	for _, delRole := range delegationRoles {
		if delRole.Name == signerDelegation {
			role = delRole
			break
		}
	}
	if role.Name == "" {
		return false, fmt.Errorf("No signer %s for repository %s", signerName, repoName)
	}
	allRoles, err := notaryRepo.ListRoles()
	if err != nil {
		return false, err
	}
	if ok, err := isLastSignerForReleases(role, allRoles); ok && !forceYes {
		removeSigner := command.PromptForConfirmation(os.Stdin, cli.Out(), fmt.Sprintf("The signer \"%s\" signed the last released version of %s. "+
			"Removing this signer will make %s unpullable. "+
			"Are you sure you want to continue?",
			signerName, repoName, repoName,
		))

		if !removeSigner {
			fmt.Fprintf(cli.Out(), "\nAborting action.\n")
			return false, nil
		}
	} else if err != nil {
		return false, err
	}
	if err = notaryRepo.RemoveDelegationKeys(releasesRoleTUFName, role.KeyIDs); err != nil {
		return false, err
	}
	if err = notaryRepo.RemoveDelegationRole(signerDelegation); err != nil {
		return false, err
	}

	if err = notaryRepo.Publish(); err != nil {
		return false, err
	}

	fmt.Fprintf(cli.Out(), "Successfully removed %s from %s\n\n", signerName, repoName)

	return true, nil
}
