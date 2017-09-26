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

func newKeyGenerateCommand(dockerCli command.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key-generate NAME [NAME...]",
		Short: "Generate and load a signing key-pair",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setupPassphraseAndGenerateKeys(dockerCli, args[0])
		},
	}
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

func setupPassphraseAndGenerateKeys(streams command.Streams, keyName string) error {
	// always use a fresh passphrase for each key generation
	freshPassRetGetter := func() notary.PassRetriever { return trust.GetPassphraseRetriever(streams.In(), streams.Out()) }
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return validateAndGenerateKey(streams, keyName, cwd, freshPassRetGetter)
}

func validateAndGenerateKey(streams command.Streams, keyName string, workingDir string, passphraseGetter func() notary.PassRetriever) error {
	if err := validateKeyArgs(keyName, workingDir); err != nil {
		return err
	}
	fmt.Fprintf(streams.Out(), "\nGenerating key for %s...\n", keyName)
	freshPassRet := passphraseGetter()
	if err := generateKey(keyName, workingDir, trust.GetTrustDirectory(), freshPassRet); err != nil {
		fmt.Fprintf(streams.Out(), err.Error())
		return fmt.Errorf("Error generating key for: %s", keyName)
	}
	pubFileName := strings.Join([]string{keyName, "pub"}, ".")
	fmt.Fprintf(streams.Out(), "Successfully generated and loaded private key. Corresponding public key available: %s\n", pubFileName)

	return nil
}

func generateKey(keyName, pubDir, privTrustDir string, passRet notary.PassRetriever) error {
	privKey, err := tufutils.GenerateKey(data.ECDSAKey)
	if err != nil {
		return err
	}

	// Automatically load the private key to local storage for use
	privKeyFileStore, err := trustmanager.NewKeyFileStore(privTrustDir, passRet)
	if err != nil {
		return err
	}

	privKeyFileStore.AddKey(trustmanager.KeyInfo{Role: data.RoleName(keyName)}, privKey)
	if err != nil {
		return err
	}

	pubKey := data.PublicKeyFromPrivate(privKey)
	pubPEM := pem.Block{
		Type: "PUBLIC KEY",
		Headers: map[string]string{
			"role": keyName,
		},
		Bytes: pubKey.Public(),
	}

	// Output the public key to a file in the CWD
	pubFileName := strings.Join([]string{keyName, "pub"}, ".")
	pubFilePath := filepath.Join(pubDir, pubFileName)
	return ioutil.WriteFile(pubFilePath, pem.EncodeToMemory(&pubPEM), notary.PrivNoExecPerms)
}
