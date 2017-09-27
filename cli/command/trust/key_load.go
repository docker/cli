package trust

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/notary"
	"github.com/docker/notary/storage"
	tufutils "github.com/docker/notary/tuf/utils"
	"github.com/docker/notary/utils"
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
		Use:   "key-load [OPTIONS] KEY",
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
	keyFileStore, err := storage.NewPrivateKeyFileStorage(filepath.Join(trustDir, notary.PrivDir), notary.KeyExtension)
	if err != nil {
		return err
	}
	privKeyImporters := []utils.Importer{keyFileStore}

	fmt.Fprintf(streams.Out(), "\nLoading key from \"%s\"...\n", keyPath)

	// Always use a fresh passphrase retriever for each import
	passRet := trust.GetPassphraseRetriever(streams.In(), streams.Out())
	if err := loadPrivKeyFromPath(privKeyImporters, keyPath, options.keyName, passRet); err != nil {
		return fmt.Errorf("error importing key from %s: %s", keyPath, err)
	}
	fmt.Fprintf(streams.Out(), "Successfully imported key from %s\n", keyPath)
	return nil
}

func loadPrivKeyFromPath(privKeyImporters []utils.Importer, keyPath, keyName string, passRet notary.PassRetriever) error {
	fileInfo, err := os.Stat(keyPath)
	if err != nil {
		return err
	}
	if fileInfo.Mode() != ownerReadOnlyPerms && fileInfo.Mode() != ownerReadAndWritePerms {
		return fmt.Errorf("private key permission from %s should be set to 400 or 600", keyPath)
	}

	from, err := os.OpenFile(keyPath, os.O_RDONLY, notary.PrivExecPerms)
	if err != nil {
		return err
	}
	defer from.Close()

	keyBytes, err := ioutil.ReadAll(from)
	if err != nil {
		return err
	}
	if _, _, err := tufutils.ExtractPrivateKeyAttributes(keyBytes); err != nil {
		return fmt.Errorf("provided file %s is not a supported private key - to add a signer's public key use docker trust signer-add", keyPath)
	}
	// Rewind the file pointer
	if _, err := from.Seek(0, 0); err != nil {
		return err
	}

	return utils.ImportKeys(from, privKeyImporters, keyName, "", passRet)
}
