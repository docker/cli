package trust

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/cli/opts"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
	tufutils "github.com/theupdateframework/notary/tuf/utils"
)

type signerAddOptions struct {
	keys   opts.ListOpts
	signer string
	repos  []string
}

func newSignerAddCommand(dockerCLI command.Cli) *cobra.Command {
	var options signerAddOptions
	cmd := &cobra.Command{
		Use:   "add OPTIONS NAME REPOSITORY [REPOSITORY...] ",
		Short: "Add a signer",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.signer = args[0]
			options.repos = args[1:]
			return addSigner(cmd.Context(), dockerCLI, options)
		},
	}
	flags := cmd.Flags()
	options.keys = opts.NewListOpts(nil)
	flags.Var(&options.keys, "key", "Path to the signer's public key file")
	return cmd
}

var validSignerName = regexp.MustCompile(`^[a-z0-9][a-z0-9\_\-]*$`).MatchString

func addSigner(ctx context.Context, dockerCLI command.Cli, options signerAddOptions) error {
	signerName := options.signer
	if !validSignerName(signerName) {
		return fmt.Errorf("signer name \"%s\" must start with lowercase alphanumeric characters and can include \"-\" or \"_\" after the first character", signerName)
	}
	if signerName == "releases" {
		return fmt.Errorf("releases is a reserved keyword, please use a different signer name")
	}

	if options.keys.Len() == 0 {
		return fmt.Errorf("path to a public key must be provided using the `--key` flag")
	}
	signerPubKeys, err := ingestPublicKeys(options.keys.GetAll())
	if err != nil {
		return err
	}
	var errRepos []string
	for _, repoName := range options.repos {
		fmt.Fprintf(dockerCLI.Out(), "Adding signer \"%s\" to %s...\n", signerName, repoName)
		if err := addSignerToRepo(ctx, dockerCLI, signerName, repoName, signerPubKeys); err != nil {
			fmt.Fprintln(dockerCLI.Err(), err.Error()+"\n")
			errRepos = append(errRepos, repoName)
		} else {
			fmt.Fprintf(dockerCLI.Out(), "Successfully added signer: %s to %s\n\n", signerName, repoName)
		}
	}
	if len(errRepos) > 0 {
		return fmt.Errorf("failed to add signer to: %s", strings.Join(errRepos, ", "))
	}
	return nil
}

func addSignerToRepo(ctx context.Context, dockerCLI command.Cli, signerName string, repoName string, signerPubKeys []data.PublicKey) error {
	imgRefAndAuth, err := trust.GetImageReferencesAndAuth(ctx, image.AuthResolver(dockerCLI), repoName)
	if err != nil {
		return err
	}

	notaryRepo, err := dockerCLI.NotaryClient(imgRefAndAuth, trust.ActionsPushAndPull)
	if err != nil {
		return trust.NotaryError(imgRefAndAuth.Reference().Name(), err)
	}

	if _, err = notaryRepo.ListTargets(); err != nil {
		switch err.(type) {
		case client.ErrRepoNotInitialized, client.ErrRepositoryNotExist:
			fmt.Fprintf(dockerCLI.Out(), "Initializing signed repository for %s...\n", repoName)
			if err := getOrGenerateRootKeyAndInitRepo(notaryRepo); err != nil {
				return trust.NotaryError(repoName, err)
			}
			fmt.Fprintf(dockerCLI.Out(), "Successfully initialized %q\n", repoName)
		default:
			return trust.NotaryError(repoName, err)
		}
	}

	newSignerRoleName := data.RoleName(path.Join(data.CanonicalTargetsRole.String(), signerName))

	if err := addStagedSigner(notaryRepo, newSignerRoleName, signerPubKeys); err != nil {
		return errors.Wrapf(err, "could not add signer to repo: %s", strings.TrimPrefix(newSignerRoleName.String(), "targets/"))
	}

	return notaryRepo.Publish()
}

func ingestPublicKeys(pubKeyPaths []string) ([]data.PublicKey, error) {
	pubKeys := []data.PublicKey{}
	for _, pubKeyPath := range pubKeyPaths {
		// Read public key bytes from PEM file, limit to 1 KiB
		pubKeyFile, err := os.OpenFile(pubKeyPath, os.O_RDONLY, 0o666)
		if err != nil {
			return nil, errors.Wrap(err, "unable to read public key from file")
		}
		defer pubKeyFile.Close()
		// limit to
		l := io.LimitReader(pubKeyFile, 1<<20)
		pubKeyBytes, err := io.ReadAll(l)
		if err != nil {
			return nil, errors.Wrap(err, "unable to read public key from file")
		}

		// Parse PEM bytes into type PublicKey
		pubKey, err := tufutils.ParsePEMPublicKey(pubKeyBytes)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse public key from file: %s", pubKeyPath)
		}
		pubKeys = append(pubKeys, pubKey)
	}
	return pubKeys, nil
}
