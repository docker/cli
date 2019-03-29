package trust

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/trust"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/tuf/data"
	tufutils "github.com/theupdateframework/notary/tuf/utils"
)

type keyGenerateOptions struct {
	name      string
	directory string
}

func newKeyGenerateCommand(dockerCli command.Streams) *cobra.Command {
	options := keyGenerateOptions{}
	cmd := &cobra.Command{
		Use:   "generate NAME",
		Short: "Generate and load a signing key-pair",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.name = args[0]
			return setupPassphraseAndGenerateKeys(dockerCli, options)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&options.directory, "dir", "", "Directory to generate key in, defaults to current directory")
	return cmd
}

// key names can use lowercase alphanumeric + _ + - characters
var validKeyName = regexp.MustCompile(`^[a-z0-9][a-z0-9\_\-]*$`).MatchString

// validate that all of the key names are unique and are alphanumeric + _ + -
// and that we do not already have public key files in the target dir on disk
func validateKeyArgs(keyName string, targetDir string) error {
	if !validKeyName(keyName) {
		return fmt.Errorf("key name \"%s\" must start with lowercase alphanumeric characters and can include \"-\" or \"_\" after the first character", keyName)
	}

	pubKeyFileName := keyName + ".pub"
	if _, err := os.Stat(targetDir); err != nil {
		return fmt.Errorf("public key path does not exist: \"%s\"", targetDir)
	}
	targetPath := filepath.Join(targetDir, pubKeyFileName)
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("public key file already exists: \"%s\"", targetPath)
	}
	return nil
}

func setupPassphraseAndGenerateKeys(streams command.Streams, opts keyGenerateOptions) error {
	targetDir := opts.directory
	if targetDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		targetDir = cwd
	}
	return validateAndGenerateKey(streams, opts.name, targetDir)
}

func validateAndGenerateKey(streams command.Streams, keyName string, workingDir string) error {
	freshPassRetGetter := func() notary.PassRetriever { return trust.GetPassphraseRetriever(streams.In(), streams.Out()) }
	if err := validateKeyArgs(keyName, workingDir); err != nil {
		return err
	}
	fmt.Fprintf(streams.Out(), "Generating key for %s...\n", keyName)
	// Automatically load the private key to local storage for use
	privKeyFileStore, err := trustmanager.NewKeyFileStore(trust.GetTrustDirectory(), freshPassRetGetter())
	if err != nil {
		return err
	}

	pubPEM, err := generateKeyAndOutputPubPEM(keyName, privKeyFileStore)
	if err != nil {
		fmt.Fprintf(streams.Out(), err.Error())
		return errors.Wrapf(err, "failed to generate key for %s", keyName)
	}

	// Output the public key to a file in the CWD or specified dir
	writtenPubFile, err := writePubKeyPEMToDir(pubPEM, keyName, workingDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(streams.Out(), "Successfully generated and loaded private key. Corresponding public key available: %s\n", writtenPubFile)

	return nil
}

func generateKeyAndOutputPubPEM(keyName string, privKeyStore trustmanager.KeyStore) (pem.Block, error) {
	privKey, err := tufutils.GenerateKey(data.ECDSAKey)
	if err != nil {
		return pem.Block{}, err
	}

	privKeyStore.AddKey(trustmanager.KeyInfo{Role: data.RoleName(keyName)}, privKey)
	if err != nil {
		return pem.Block{}, err
	}

	pubKey := data.PublicKeyFromPrivate(privKey)
	return pem.Block{
		Type: "PUBLIC KEY",
		Headers: map[string]string{
			"role": keyName,
		},
		Bytes: pubKey.Public(),
	}, nil
}

func writePubKeyPEMToDir(pubPEM pem.Block, keyName, workingDir string) (string, error) {
	// Output the public key to a file in the CWD or specified dir
	pubFileName := strings.Join([]string{keyName, "pub"}, ".")
	pubFilePath := filepath.Join(workingDir, pubFileName)
	if err := ioutil.WriteFile(pubFilePath, pem.EncodeToMemory(&pubPEM), notary.PrivNoExecPerms); err != nil {
		return "", errors.Wrapf(err, "failed to write public key to %s", pubFilePath)
	}
	return pubFilePath, nil
}
