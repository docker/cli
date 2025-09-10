package context

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

func newImportCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import CONTEXT FILE|-",
		Short: "Import a context from a tar or zip file",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(dockerCLI, args[0], args[1])
		},
		// TODO(thaJeztah): this should also include "-"
		ValidArgsFunction:     completion.FileNames(),
		DisableFlagsInUseLine: true,
	}
	return cmd
}

// runImport imports a Docker context.
func runImport(dockerCLI command.Cli, name string, source string) error {
	if err := checkContextNameForCreation(dockerCLI.ContextStore(), name); err != nil {
		return err
	}

	var reader io.Reader
	if source == "-" {
		reader = dockerCLI.In()
	} else {
		f, err := os.Open(source)
		if err != nil {
			return err
		}
		defer f.Close()
		reader = f
	}

	if err := store.Import(name, dockerCLI.ContextStore(), reader); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	_, _ = fmt.Fprintf(dockerCLI.Err(), "Successfully imported context %q\n", name)
	return nil
}
