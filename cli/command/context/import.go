package context

import (
	"archive/tar"
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

func newImportCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import CONTEXT FILE|-",
		Short: "Import a context from a tar file",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunImport(dockerCli, args[0], args[1])
		},
	}
	return cmd
}

// RunImport imports a Docker context
func RunImport(dockerCli command.Cli, name string, source string) error {
	if err := checkContextNameForCreation(dockerCli.ContextStore(), name); err != nil {
		return err
	}
	var reader io.Reader
	if source == "-" {
		reader = dockerCli.In()
	} else {
		f, err := os.Open(source)
		if err != nil {
			return err
		}
		defer f.Close()
		reader = f
	}
	if err := store.Import(name, dockerCli.ContextStore(), reader); err != nil {
		if err == tar.ErrHeader {
			// try with ucp bundle file logic, if it fails, return the original error
			if err := importBundleFile(dockerCli, name, source); err == nil {
				return nil
			}
		}
		return err
	}
	fmt.Fprintln(dockerCli.Out(), name)
	fmt.Fprintf(dockerCli.Err(), "Successfully imported context %q\n", name)
	return nil
}

func importBundleFile(dockerCli command.Cli, name string, source string) error {
	zipArchive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer zipArchive.Close()
	for _, f := range zipArchive.File {
		if strings.HasSuffix(f.Name, ".dockercontext") {
			reader, err := f.Open()
			if err != nil {
				return err
			}
			defer reader.Close()
			return store.Import(name, dockerCli.ContextStore(), reader)
		}
	}
	return errors.New("context not found in zip file")
}
