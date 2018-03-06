package trust

import (
	"context"
	"fmt"
	"io"
	"path"
	"sort"
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

type signOptions struct {
	local     bool
	imageName string
}

func newSignCommand(dockerCli command.Cli) *cobra.Command {
	options := signOptions{}
	cmd := &cobra.Command{
		Use:   "sign IMAGE:TAG",
		Short: "Sign an image",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.imageName = args[0]
			return runSignImage(dockerCli, options)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&options.local, "local", false, "Sign a locally tagged image")
	return cmd
}

func runSignImage(cli command.Cli, options signOptions) error {
	imageName := options.imageName
	ctx := context.Background()
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, nil, image.AuthResolver(cli), imageName)
	if err != nil {
		return err
	}
	if err := validateTag(imgRefAndAuth); err != nil {
		return err
	}

	notaryRepo, err := cli.NotaryClient(imgRefAndAuth, trust.ActionsPushAndPull)
	if err != nil {
		return trust.NotaryError(imgRefAndAuth.Reference().Name(), err)
	}
	if err = clearChangeList(notaryRepo); err != nil {
		return err
	}
	defer clearChangeList(notaryRepo)

	// get the latest repository metadata so we can figure out which roles to sign
	if _, err = notaryRepo.ListTargets(); err != nil {
		switch err.(type) {
		case client.ErrRepoNotInitialized, client.ErrRepositoryNotExist:
			// before initializing a new repo, check that the image exists locally:
			if err := checkLocalImageExistence(ctx, cli, imageName); err != nil {
				return err
			}

			userRole := data.RoleName(path.Join(data.CanonicalTargetsRole.String(), imgRefAndAuth.AuthConfig().Username))
			if err := initNotaryRepoWithSigners(notaryRepo, userRole); err != nil {
				return trust.NotaryError(imgRefAndAuth.Reference().Name(), err)
			}

			fmt.Fprintf(cli.Out(), "Created signer: %s\n", imgRefAndAuth.AuthConfig().Username)
			fmt.Fprintf(cli.Out(), "Finished initializing signed repository for %s\n", imageName)
		default:
			return trust.NotaryError(imgRefAndAuth.RepoInfo().Name.Name(), err)
		}
	}
	requestPrivilege := command.RegistryAuthenticationPrivilegedFunc(cli, imgRefAndAuth.RepoInfo().Index, "push")
	target, err := createTarget(notaryRepo, imgRefAndAuth.Tag())
	if err != nil || options.local {
		switch err := err.(type) {
		// If the error is nil then the local flag is set
		case client.ErrNoSuchTarget, client.ErrRepositoryNotExist, nil:
			// Fail fast if the image doesn't exist locally
			if err := checkLocalImageExistence(ctx, cli, imageName); err != nil {
				return err
			}
			fmt.Fprintf(cli.Err(), "Signing and pushing trust data for local image %s, may overwrite remote trust data\n", imageName)
			return image.TrustedPush(ctx, cli, imgRefAndAuth.RepoInfo(), imgRefAndAuth.Reference(), *imgRefAndAuth.AuthConfig(), requestPrivilege)
		default:
			return err
		}
	}
	return signAndPublishToTarget(cli.Out(), imgRefAndAuth, notaryRepo, target)
}

func signAndPublishToTarget(out io.Writer, imgRefAndAuth trust.ImageRefAndAuth, notaryRepo client.Repository, target client.Target) error {
	tag := imgRefAndAuth.Tag()
	fmt.Fprintf(out, "Signing and pushing trust metadata for %s\n", imgRefAndAuth.Name())
	existingSigInfo, err := getExistingSignatureInfoForReleasedTag(notaryRepo, tag)
	if err != nil {
		return err
	}
	err = image.AddTargetToAllSignableRoles(notaryRepo, &target)
	if err == nil {
		prettyPrintExistingSignatureInfo(out, existingSigInfo)
		err = notaryRepo.Publish()
	}
	if err != nil {
		return errors.Wrapf(err, "failed to sign %s:%s", imgRefAndAuth.RepoInfo().Name.Name(), tag)
	}
	fmt.Fprintf(out, "Successfully signed %s:%s\n", imgRefAndAuth.RepoInfo().Name.Name(), tag)
	return nil
}

func validateTag(imgRefAndAuth trust.ImageRefAndAuth) error {
	tag := imgRefAndAuth.Tag()
	if tag == "" {
		if imgRefAndAuth.Digest() != "" {
			return fmt.Errorf("cannot use a digest reference for IMAGE:TAG")
		}
		return fmt.Errorf("No tag specified for %s", imgRefAndAuth.Name())
	}
	return nil
}

func checkLocalImageExistence(ctx context.Context, cli command.Cli, imageName string) error {
	_, _, err := cli.Client().ImageInspectWithRaw(ctx, imageName)
	return err
}

func createTarget(notaryRepo client.Repository, tag string) (client.Target, error) {
	target := &client.Target{}
	var err error
	if tag == "" {
		return *target, fmt.Errorf("No tag specified")
	}
	target.Name = tag
	target.Hashes, target.Length, err = getSignedManifestHashAndSize(notaryRepo, tag)
	return *target, err
}

func getSignedManifestHashAndSize(notaryRepo client.Repository, tag string) (data.Hashes, int64, error) {
	targets, err := notaryRepo.GetAllTargetMetadataByName(tag)
	if err != nil {
		return nil, 0, err
	}
	return getReleasedTargetHashAndSize(targets, tag)
}

func getReleasedTargetHashAndSize(targets []client.TargetSignedStruct, tag string) (data.Hashes, int64, error) {
	for _, tgt := range targets {
		if isReleasedTarget(tgt.Role.Name) {
			return tgt.Target.Hashes, tgt.Target.Length, nil
		}
	}
	return nil, 0, client.ErrNoSuchTarget(tag)
}

func getExistingSignatureInfoForReleasedTag(notaryRepo client.Repository, tag string) (trustTagRow, error) {
	targets, err := notaryRepo.GetAllTargetMetadataByName(tag)
	if err != nil {
		return trustTagRow{}, err
	}
	releasedTargetInfoList := matchReleasedSignatures(targets)
	if len(releasedTargetInfoList) == 0 {
		return trustTagRow{}, nil
	}
	return releasedTargetInfoList[0], nil
}

func prettyPrintExistingSignatureInfo(out io.Writer, existingSigInfo trustTagRow) {
	sort.Strings(existingSigInfo.Signers)
	joinedSigners := strings.Join(existingSigInfo.Signers, ", ")
	fmt.Fprintf(out, "Existing signatures for tag %s digest %s from:\n%s\n", existingSigInfo.SignedTag, existingSigInfo.Digest, joinedSigners)
}

func initNotaryRepoWithSigners(notaryRepo client.Repository, newSigner data.RoleName) error {
	rootKey, err := getOrGenerateNotaryKey(notaryRepo, data.CanonicalRootRole)
	if err != nil {
		return err
	}
	rootKeyID := rootKey.ID()

	// Initialize the notary repository with a remotely managed snapshot key
	if err := notaryRepo.Initialize([]string{rootKeyID}, data.CanonicalSnapshotRole); err != nil {
		return err
	}

	signerKey, err := getOrGenerateNotaryKey(notaryRepo, newSigner)
	if err != nil {
		return err
	}
	if err := addStagedSigner(notaryRepo, newSigner, []data.PublicKey{signerKey}); err != nil {
		return errors.Wrapf(err, "could not add signer to repo: %s", strings.TrimPrefix(newSigner.String(), "targets/"))
	}

	return notaryRepo.Publish()
}

// generates an ECDSA key without a GUN for the specified role
func getOrGenerateNotaryKey(notaryRepo client.Repository, role data.RoleName) (data.PublicKey, error) {
	// use the signer name in the PEM headers if this is a delegation key
	if data.IsDelegation(role) {
		role = data.RoleName(notaryRoleToSigner(role))
	}
	keys := notaryRepo.GetCryptoService().ListKeys(role)
	var err error
	var key data.PublicKey
	// always select the first key by ID
	if len(keys) > 0 {
		sort.Strings(keys)
		keyID := keys[0]
		privKey, _, err := notaryRepo.GetCryptoService().GetPrivateKey(keyID)
		if err != nil {
			return nil, err
		}
		key = data.PublicKeyFromPrivate(privKey)
	} else {
		key, err = notaryRepo.GetCryptoService().Create(role, "", data.ECDSAKey)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
}

// stages changes to add a signer with the specified name and key(s).  Adds to targets/<name> and targets/releases
func addStagedSigner(notaryRepo client.Repository, newSigner data.RoleName, signerKeys []data.PublicKey) error {
	// create targets/<username>
	if err := notaryRepo.AddDelegationRoleAndKeys(newSigner, signerKeys); err != nil {
		return err
	}
	if err := notaryRepo.AddDelegationPaths(newSigner, []string{""}); err != nil {
		return err
	}

	// create targets/releases
	if err := notaryRepo.AddDelegationRoleAndKeys(trust.ReleasesRole, signerKeys); err != nil {
		return err
	}
	return notaryRepo.AddDelegationPaths(trust.ReleasesRole, []string{""})
}
