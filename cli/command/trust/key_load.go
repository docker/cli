package trust

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/notary"
	"github.com/docker/notary/storage"
	"github.com/docker/notary/trustmanager"
	tufutils "github.com/docker/notary/tuf/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	ownerReadOnlyPerms     = 0400
	ownerReadAndWritePerms = 0600
)

type keyLoadOptions struct {
	keyName string
}

func newKeyLoadCommand(dockerCli command.Streams) *cobra.Command {
	var options keyLoadOptions
	cmd := &cobra.Command{
		Use:   "load [OPTIONS] KEY",
		Short: "Load a private key file for signing",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return loadPrivKey(dockerCli, args[0], options)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&options.keyName, "name", "n", "signer", "Name for the loaded key")
	return cmd
}

func loadPrivKey(streams command.Streams, keyPath string, options keyLoadOptions) error {
	trustDir := trust.GetTrustDirectory()
	keyFileStore, err := storage.NewPrivateKeyFileStorage(trustDir, notary.KeyExtension)
	if err != nil {
		return err
	}
	privKeyImporters := []trustmanager.Importer{keyFileStore}

	fmt.Fprintf(streams.Out(), "\nLoading key from \"%s\"...\n", keyPath)

	// Always use a fresh passphrase retriever for each import
	passRet := trust.GetPassphraseRetriever(streams.In(), streams.Out())
	keyBytes, err := getPrivKeyBytesFromPath(keyPath)
	if err != nil {
		return errors.Wrapf(err, "error reading key from %s", keyPath)
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
	if fileInfo.Mode() != ownerReadOnlyPerms && fileInfo.Mode() != ownerReadAndWritePerms {
		return nil, fmt.Errorf("private key permission from %s should be set to 400 or 600", keyPath)
	}

	from, err := os.OpenFile(keyPath, os.O_RDONLY, notary.PrivExecPerms)
	if err != nil {
		return nil, err
	}
	defer from.Close()

	return ioutil.ReadAll(from)
}

func loadPrivKeyBytesToStore(privKeyBytes []byte, privKeyImporters []trustmanager.Importer, keyPath, keyName string, passRet notary.PassRetriever) error {
	if _, _, err := tufutils.ExtractPrivateKeyAttributes(privKeyBytes); err != nil {
		return fmt.Errorf("provided file %s is not a supported private key - to add a signer's public key use docker trust signer add", keyPath)
	}
	// Make a reader, rewind the file pointer
	return trustmanager.ImportKeys(bytes.NewReader(privKeyBytes), privKeyImporters, keyName, "", passRet)
}
