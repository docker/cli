package context

import (
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
		Short: "Import a context using a tar or zip file",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunImport(dockerCli, args[0], args[1])
		},
	}
	return cmd
}

func getReaderAndImportType(dockerCli command.Cli, source string) (io.Reader, store.ImportType, func(), error) {
	var (
		reader     io.Reader
		importType store.ImportType
		cleanup    func()
	)

	if source == "-" {
		reader = dockerCli.In()
		importType = store.Cli
		cleanup = func() {}
	} else {
		if strings.HasSuffix(source, ".zip") {
			importType = store.Zip
		} else {
			importType = store.Tar
		}

		f, err := os.Open(source)
		if err != nil {
			return nil, importType, nil, err
		}

		cleanup = func() {
			f.Close()
		}
		reader = f
	}

	return reader, importType, cleanup, nil
}

// RunImport imports a Docker context
func RunImport(dockerCli command.Cli, name string, source string) error {
	if err := checkContextNameForCreation(dockerCli.ContextStore(), name); err != nil {
		return err
	}

	reader, importType, cleanup, err := getReaderAndImportType(dockerCli, source)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := store.Import(name, dockerCli.ContextStore(), reader, importType); err != nil {
		return err
	}
	fmt.Fprintln(dockerCli.Out(), name)
	fmt.Fprintf(dockerCli.Err(), "Successfully imported context %q\n", name)
	return nil
}
