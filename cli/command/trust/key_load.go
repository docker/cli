package trust

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/trust"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/storage"
	"github.com/theupdateframework/notary/trustmanager"
	tufutils "github.com/theupdateframework/notary/tuf/utils"
)

const (
	nonOwnerReadWriteMask = 0077
)

type keyLoadOptions struct {
	keyName string
}

func newKeyLoadCommand(dockerCli command.Streams) *cobra.Command {
	var options keyLoadOptions
	cmd := &cobra.Command{
		Use:   "load [OPTIONS] KEYFILE",
		Short: "Load a private key file for signing",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return loadPrivKey(dockerCli, args[0], options)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&options.keyName, "name", "signer", "Name for the loaded key")
	return cmd
}

func loadPrivKey(streams command.Streams, keyPath string, options keyLoadOptions) error {
	// validate the key name if provided
	if options.keyName != "" && !validKeyName(options.keyName) {
		return fmt.Errorf("key name \"%s\" must start with lowercase alphanumeric characters and can include \"-\" or \"_\" after the first character", options.keyName)
	}
	trustDir := trust.GetTrustDirectory()
	keyFileStore, err := storage.NewPrivateKeyFileStorage(trustDir, notary.KeyExtension)
	if err != nil {
		return err
	}
	privKeyImporters := []trustmanager.Importer{keyFileStore}

	fmt.Fprintf(streams.Out(), "Loading key from \"%s\"...\n", keyPath)

	// Always use a fresh passphrase retriever for each import
	passRet := trust.GetPassphraseRetriever(streams.In(), streams.Out())
	keyBytes, err := getPrivKeyBytesFromPath(keyPath)
	if err != nil {
		return errors.Wrapf(err, "refusing to load key from %s", keyPath)
	}
	if err := loadPrivKeyBytesToStore(keyBytes, privKeyImporters, keyPath, options.keyName, passRet); err != nil {
		return errors.Wrapf(err, "error importing key from %s", keyPath)
	}
	fmt.Fprintf(streams.Out(), "Successfully imported key from %s\n", keyPath)
	return nil
}

func getPrivKeyBytesFromPath(keyPath string) ([]byte, error) {
	fileInfo, err := os.Stat(keyPath)
	if err != nil {
		return nil, err
	}
	if fileInfo.Mode()&nonOwnerReadWriteMask != 0 {
		return nil, fmt.Errorf("private key file %s must not be readable or writable by others", keyPath)
	}

	from, err := os.OpenFile(keyPath, os.O_RDONLY, notary.PrivExecPerms)
	if err != nil {
		return nil, err
	}
	defer from.Close()

	return ioutil.ReadAll(from)
}

func loadPrivKeyBytesToStore(privKeyBytes []byte, privKeyImporters []trustmanager.Importer, keyPath, keyName string, passRet notary.PassRetriever) error {
	var err error
	if _, _, err = tufutils.ExtractPrivateKeyAttributes(privKeyBytes); err != nil {
		return fmt.Errorf("provided file %s is not a supported private key - to add a signer's public key use docker trust signer add", keyPath)
	}
	if privKeyBytes, err = decodePrivKeyIfNecessary(privKeyBytes, passRet); err != nil {
		return errors.Wrapf(err, "cannot load key from provided file %s", keyPath)
	}
	// Make a reader, rewind the file pointer
	return trustmanager.ImportKeys(bytes.NewReader(privKeyBytes), privKeyImporters, keyName, "", passRet)
}

func decodePrivKeyIfNecessary(privPemBytes []byte, passRet notary.PassRetriever) ([]byte, error) {
	pemBlock, _ := pem.Decode(privPemBytes)
	_, containsDEKInfo := pemBlock.Headers["DEK-Info"]
	if containsDEKInfo || pemBlock.Type == "ENCRYPTED PRIVATE KEY" {
		// if we do not have enough information to properly import, try to decrypt the key
		if _, ok := pemBlock.Headers["path"]; !ok {
			privKey, _, err := trustmanager.GetPasswdDecryptBytes(passRet, privPemBytes, "", "encrypted")
			if err != nil {
				return []byte{}, fmt.Errorf("could not decrypt key")
			}
			privPemBytes = privKey.Private()
		}
	}
	return privPemBytes, nil
}
