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
	"github.com/docker/notary"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
	tufutils "github.com/docker/notary/tuf/utils"
	"github.com/spf13/cobra"
)

type keyGenerateOptions struct {
	name      string
	directory string
}

func newKeyGenerateCommand(dockerCli command.Streams) *cobra.Command {
	options := keyGenerateOptions{}
	cmd := &cobra.Command{
		Use:   "generate NAME [NAME...]",
		Short: "Generate and load a signing key-pair",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.name = args[0]
			return setupPassphraseAndGenerateKeys(dockerCli, options)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&options.directory, "dir", "d", "", "Directory to generate key in, defaults to current directory")
	return cmd
}

// key names can use alphanumeric + _ + - characters
var validKeyName = regexp.MustCompile(`^[a-zA-Z0-9\_]+[a-zA-Z0-9\_\-]*$`).MatchString

// validate that all of the key names are unique and are alphanumeric + _ + -
// and that we do not already have public key files in the current dir on disk
func validateKeyArgs(keyName string, cwdPath string) error {
	if !validKeyName(keyName) {
		return fmt.Errorf("key name \"%s\" must not contain special characters", keyName)
	}

	pubKeyFileName := keyName + ".pub"
	if _, err := os.Stat(filepath.Join(cwdPath, pubKeyFileName)); err == nil {
		return fmt.Errorf("public key file already exists: \"%s\"", pubKeyFileName)
	}
	return nil
}

func setupPassphraseAndGenerateKeys(streams command.Streams, opts keyGenerateOptions) error {
	targetDir := opts.directory
	// if the target dir is empty, default to CWD
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
	fmt.Fprintf(streams.Out(), "\nGenerating key for %s...\n", keyName)
	// Automatically load the private key to local storage for use
	privKeyFileStore, err := trustmanager.NewKeyFileStore(trust.GetTrustDirectory(), freshPassRetGetter())
	if err != nil {
		return err
	}

	pubPEM, err := generateKeyAndOutputPubPEM(keyName, privKeyFileStore)
	if err != nil {
		fmt.Fprintf(streams.Out(), err.Error())
		return fmt.Errorf("Error generating key for: %s", keyName)
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
		return "", fmt.Errorf("Error writing public key to location: %s", pubFilePath)
	}
	return pubFilePath, nil
}
