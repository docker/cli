package trust

import (
	"context"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/trust"
	imagetypes "github.com/docker/docker/api/types/image"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	notaryclient "github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
)

type signOptions struct {
	local     bool
	imageName string
}

func newSignCommand(dockerCLI command.Cli) *cobra.Command {
	options := signOptions{}
	cmd := &cobra.Command{
		Use:   "sign IMAGE:TAG",
		Short: "Sign an image",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.imageName = args[0]
			return runSignImage(cmd.Context(), dockerCLI, options)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&options.local, "local", false, "Sign a locally tagged image")
	return cmd
}

func runSignImage(ctx context.Context, dockerCLI command.Cli, options signOptions) error {
	imageName := options.imageName
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, image.AuthResolver(dockerCLI), imageName)
	if err != nil {
		return err
	}
	if err := validateTag(imgRefAndAuth); err != nil {
		return err
	}

	notaryRepo, err := newNotaryClient(dockerCLI, imgRefAndAuth, trust.ActionsPushAndPull)
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
		case notaryclient.ErrRepoNotInitialized, notaryclient.ErrRepositoryNotExist:
			// before initializing a new repo, check that the image exists locally:
			if err := checkLocalImageExistence(ctx, dockerCLI.Client(), imageName); err != nil {
				return err
			}

			userRole := data.RoleName(path.Join(data.CanonicalTargetsRole.String(), imgRefAndAuth.AuthConfig().Username))
			if err := initNotaryRepoWithSigners(notaryRepo, userRole); err != nil {
				return trust.NotaryError(imgRefAndAuth.Reference().Name(), err)
			}

			_, _ = fmt.Fprintln(dockerCLI.Out(), "Created signer:", imgRefAndAuth.AuthConfig().Username)
			_, _ = fmt.Fprintln(dockerCLI.Out(), "Finished initializing signed repository for", imageName)
		default:
			return trust.NotaryError(imgRefAndAuth.RepoInfo().Name.Name(), err)
		}
	}
	var requestPrivilege registrytypes.RequestAuthConfig
	if dockerCLI.In().IsTerminal() {
		requestPrivilege = command.RegistryAuthenticationPrivilegedFunc(dockerCLI, imgRefAndAuth.RepoInfo().Index, "push")
	}
	target, err := createTarget(notaryRepo, imgRefAndAuth.Tag())
	if err != nil || options.local {
		switch err := err.(type) {
		// If the error is nil then the local flag is set
		case notaryclient.ErrNoSuchTarget, notaryclient.ErrRepositoryNotExist, nil:
			// Fail fast if the image doesn't exist locally
			if err := checkLocalImageExistence(ctx, dockerCLI.Client(), imageName); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(dockerCLI.Err(), "Signing and pushing trust data for local image %s, may overwrite remote trust data\n", imageName)

			authConfig := command.ResolveAuthConfig(dockerCLI.ConfigFile(), imgRefAndAuth.RepoInfo().Index)
			encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
			if err != nil {
				return err
			}
			responseBody, err := dockerCLI.Client().ImagePush(ctx, reference.FamiliarString(imgRefAndAuth.Reference()), imagetypes.PushOptions{
				RegistryAuth:  encodedAuth,
				PrivilegeFunc: requestPrivilege,
			})
			if err != nil {
				return err
			}
			defer responseBody.Close()
			return trust.PushTrustedReference(ctx, dockerCLI, imgRefAndAuth.RepoInfo(), imgRefAndAuth.Reference(), authConfig, responseBody, command.UserAgent())
		default:
			return err
		}
	}
	return signAndPublishToTarget(dockerCLI.Out(), imgRefAndAuth, notaryRepo, target)
}

func signAndPublishToTarget(out io.Writer, imgRefAndAuth trust.ImageRefAndAuth, notaryRepo notaryclient.Repository, target notaryclient.Target) error {
	tag := imgRefAndAuth.Tag()
	_, _ = fmt.Fprintln(out, "Signing and pushing trust metadata for", imgRefAndAuth.Name())
	existingSigInfo, err := getExistingSignatureInfoForReleasedTag(notaryRepo, tag)
	if err != nil {
		return err
	}
	err = trust.AddToAllSignableRoles(notaryRepo, &target)
	if err == nil {
		prettyPrintExistingSignatureInfo(out, existingSigInfo)
		err = notaryRepo.Publish()
	}
	if err != nil {
		return errors.Wrapf(err, "failed to sign %s:%s", imgRefAndAuth.RepoInfo().Name.Name(), tag)
	}
	_, _ = fmt.Fprintf(out, "Successfully signed %s:%s\n", imgRefAndAuth.RepoInfo().Name.Name(), tag)
	return nil
}

func validateTag(imgRefAndAuth trust.ImageRefAndAuth) error {
	tag := imgRefAndAuth.Tag()
	if tag == "" {
		if imgRefAndAuth.Digest() != "" {
			return errors.New("cannot use a digest reference for IMAGE:TAG")
		}
		return fmt.Errorf("no tag specified for %s", imgRefAndAuth.Name())
	}
	return nil
}

func checkLocalImageExistence(ctx context.Context, apiClient client.APIClient, imageName string) error {
	_, err := apiClient.ImageInspect(ctx, imageName)
	return err
}

func createTarget(notaryRepo notaryclient.Repository, tag string) (notaryclient.Target, error) {
	target := &notaryclient.Target{}
	var err error
	if tag == "" {
		return *target, errors.New("no tag specified")
	}
	target.Name = tag
	target.Hashes, target.Length, err = getSignedManifestHashAndSize(notaryRepo, tag)
	return *target, err
}

func getSignedManifestHashAndSize(notaryRepo notaryclient.Repository, tag string) (data.Hashes, int64, error) {
	targets, err := notaryRepo.GetAllTargetMetadataByName(tag)
	if err != nil {
		return nil, 0, err
	}
	return getReleasedTargetHashAndSize(targets, tag)
}

func getReleasedTargetHashAndSize(targets []notaryclient.TargetSignedStruct, tag string) (data.Hashes, int64, error) {
	for _, tgt := range targets {
		if isReleasedTarget(tgt.Role.Name) {
			return tgt.Target.Hashes, tgt.Target.Length, nil
		}
	}
	return nil, 0, notaryclient.ErrNoSuchTarget(tag)
}

func getExistingSignatureInfoForReleasedTag(notaryRepo notaryclient.Repository, tag string) (trustTagRow, error) {
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
	_, _ = fmt.Fprintf(out, "Existing signatures for tag %s digest %s from:\n%s\n", existingSigInfo.SignedTag, existingSigInfo.Digest, joinedSigners)
}

func initNotaryRepoWithSigners(notaryRepo notaryclient.Repository, newSigner data.RoleName) error {
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
func getOrGenerateNotaryKey(notaryRepo notaryclient.Repository, role data.RoleName) (data.PublicKey, error) {
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
func addStagedSigner(notaryRepo notaryclient.Repository, newSigner data.RoleName, signerKeys []data.PublicKey) error {
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
