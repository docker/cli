package context

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

func newImportCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import CONTEXT FILE|-",
		Short: "Import a context from a tar or zip file",
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
		return err
	}

	fmt.Fprintln(dockerCli.Out(), name)
	fmt.Fprintf(dockerCli.Err(), "Successfully imported context %q\n", name)
	return nil
}
