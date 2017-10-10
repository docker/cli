package trust

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/notary/client"
	"github.com/docker/notary/tuf/data"
	tufutils "github.com/docker/notary/tuf/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type signerAddOptions struct {
	keys   opts.ListOpts
	signer string
	images []string
}

func newSignerAddCommand(dockerCli command.Cli) *cobra.Command {
	var options signerAddOptions
	cmd := &cobra.Command{
		Use:   "add [OPTIONS] NAME IMAGE [IMAGE...] ",
		Short: "Add a signer",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.signer = args[0]
			options.images = args[1:]
			return addSigner(dockerCli, options)
		},
	}
	flags := cmd.Flags()
	options.keys = opts.NewListOpts(nil)
	flags.VarP(&options.keys, "key", "k", "Path to the signer's public key(s)")
	return cmd
}

var validSignerName = regexp.MustCompile(`^[a-z0-9]+[a-z0-9\_\-]*$`).MatchString

func addSigner(cli command.Cli, options signerAddOptions) error {
	signerName := options.signer
	if !validSignerName(signerName) {
		return fmt.Errorf("signer name \"%s\" must not contain uppercase or special characters", signerName)
	}
	if signerName == "releases" {
		return fmt.Errorf("releases is a reserved keyword, please use a different signer name")
	}

	if options.keys.Len() < 1 {
		return fmt.Errorf("path to a valid public key must be provided using the `--key` flag")
	}

	var errImages []string
	for _, imageName := range options.images {
		if err := addSignerToImage(cli, signerName, imageName, options.keys.GetAll()); err != nil {
			fmt.Fprintln(cli.Out(), err.Error())
			errImages = append(errImages, imageName)
		} else {
			fmt.Fprintf(cli.Out(), "Successfully added signer: %s to %s\n", signerName, imageName)
		}
	}
	if len(errImages) > 0 {
		return fmt.Errorf("Failed to add signer to: %s", strings.Join(errImages, ", "))
	}
	return nil
}

func addSignerToImage(cli command.Cli, signerName string, imageName string, keyPaths []string) error {
	fmt.Fprintf(cli.Out(), "\nAdding signer \"%s\" to %s...\n", signerName, imageName)

	ctx := context.Background()
	authResolver := func(ctx context.Context, index *registrytypes.IndexInfo) types.AuthConfig {
		return command.ResolveAuthConfig(ctx, cli, index)
	}
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, authResolver, imageName)
	if err != nil {
		return err
	}

	notaryRepo, err := cli.NotaryClient(*imgRefAndAuth, trust.ActionsPushAndPull)
	if err != nil {
		return trust.NotaryError(imgRefAndAuth.Reference().Name(), err)
	}

	if _, err = notaryRepo.ListTargets(); err != nil {
		switch err.(type) {
		case client.ErrRepoNotInitialized, client.ErrRepositoryNotExist:
			fmt.Fprintf(cli.Out(), "Initializing signed repository for %s...\n", imageName)
			if err := getOrGenerateRootKeyAndInitRepo(notaryRepo); err != nil {
				return trust.NotaryError(imageName, err)
			}
			fmt.Fprintf(cli.Out(), "Successfully initialized %q\n", imageName)
		default:
			return trust.NotaryError(imageName, err)
		}
	}

	newSignerRoleName := data.RoleName(path.Join(data.CanonicalTargetsRole.String(), signerName))

	signerPubKeys, err := ingestPublicKeys(keyPaths)
	if err != nil {
		return err
	}
	if err := addStagedSigner(notaryRepo, newSignerRoleName, signerPubKeys); err != nil {
		return errors.Wrapf(err, "could not add signer to repo: %s", strings.TrimPrefix(newSignerRoleName.String(), "targets/"))
	}

	return notaryRepo.Publish()
}

func ingestPublicKeys(pubKeyPaths []string) ([]data.PublicKey, error) {
	pubKeys := []data.PublicKey{}
	for _, pubKeyPath := range pubKeyPaths {
		// Read public key bytes from PEM file
		pubKeyBytes, err := ioutil.ReadFile(pubKeyPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("file for public key does not exist: %s", pubKeyPath)
			}
			return nil, fmt.Errorf("unable to read public key from file: %s", pubKeyPath)
		}

		// Parse PEM bytes into type PublicKey
		pubKey, err := tufutils.ParsePEMPublicKey(pubKeyBytes)
		if err != nil {
			return nil, err
		}
		pubKeys = append(pubKeys, pubKey)
	}
	return pubKeys, nil
}
