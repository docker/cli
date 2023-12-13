package trust

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/trust"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
)

type revokeOptions struct {
	forceYes bool
}

func newRevokeCommand(dockerCLI command.Cli) *cobra.Command {
	options := revokeOptions{}
	cmd := &cobra.Command{
		Use:   "revoke [OPTIONS] IMAGE[:TAG]",
		Short: "Remove trust for an image",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return revokeTrust(dockerCLI, args[0], options)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&options.forceYes, "yes", "y", false, "Do not prompt for confirmation")
	return cmd
}

func revokeTrust(dockerCLI command.Cli, remote string, options revokeOptions) error {
	ctx := context.Background()
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, image.AuthResolver(dockerCLI), remote)
	if err != nil {
		return err
	}
	tag := imgRefAndAuth.Tag()
	if imgRefAndAuth.Tag() == "" && imgRefAndAuth.Digest() != "" {
		return fmt.Errorf("cannot use a digest reference for IMAGE:TAG")
	}
	if imgRefAndAuth.Tag() == "" && !options.forceYes {
		deleteRemote := command.PromptForConfirmation(os.Stdin, dockerCLI.Out(), fmt.Sprintf("Please confirm you would like to delete all signature data for %s?", remote))
		if !deleteRemote {
			fmt.Fprintf(dockerCLI.Out(), "\nAborting action.\n")
			return nil
		}
	}

	notaryRepo, err := dockerCLI.NotaryClient(imgRefAndAuth, trust.ActionsPushAndPull)
	if err != nil {
		return err
	}

	if err = clearChangeList(notaryRepo); err != nil {
		return err
	}
	defer clearChangeList(notaryRepo)
	if err := revokeSignature(notaryRepo, tag); err != nil {
		return fmt.Errorf("could not remove signature for %s: %w", remote, err)
	}
	fmt.Fprintf(dockerCLI.Out(), "Successfully deleted signature for %s\n", remote)
	return nil
}

func revokeSignature(notaryRepo client.Repository, tag string) error {
	if tag != "" {
		// Revoke signature for the specified tag
		if err := revokeSingleSig(notaryRepo, tag); err != nil {
			return err
		}
	} else {
		// revoke all signatures for the image, as no tag was given
		if err := revokeAllSigs(notaryRepo); err != nil {
			return err
		}
	}

	//  Publish change
	return notaryRepo.Publish()
}

func revokeSingleSig(notaryRepo client.Repository, tag string) error {
	releasedTargetWithRole, err := notaryRepo.GetTargetByName(tag, trust.ReleasesRole, data.CanonicalTargetsRole)
	if err != nil {
		return err
	}
	releasedTarget := releasedTargetWithRole.Target
	return getSignableRolesForTargetAndRemove(releasedTarget, notaryRepo)
}

func revokeAllSigs(notaryRepo client.Repository) error {
	releasedTargetWithRoleList, err := notaryRepo.ListTargets(trust.ReleasesRole, data.CanonicalTargetsRole)
	if err != nil {
		return err
	}

	if len(releasedTargetWithRoleList) == 0 {
		return fmt.Errorf("no signed tags to remove")
	}

	// we need all the roles that signed each released target so we can remove from all roles.
	for _, releasedTargetWithRole := range releasedTargetWithRoleList {
		// remove from all roles
		if err := getSignableRolesForTargetAndRemove(releasedTargetWithRole.Target, notaryRepo); err != nil {
			return err
		}
	}
	return nil
}

// get all the roles that signed the target and removes it from all roles.
func getSignableRolesForTargetAndRemove(releasedTarget client.Target, notaryRepo client.Repository) error {
	signableRoles, err := trust.GetSignableRoles(notaryRepo, &releasedTarget)
	if err != nil {
		return err
	}
	// remove from all roles
	return notaryRepo.RemoveTarget(releasedTarget.Name, signableRoles...)
}
