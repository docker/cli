package context

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/cli/internal/hint"
	"github.com/spf13/cobra"
)

func newExportCommand(dockerCLI command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "export [OPTIONS] CONTEXT [FILE|-]",
		Short: "Export a context to a tar archive FILE or a tar stream on STDOUT.",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			contextName := args[0]
			var dest string
			if len(args) == 2 {
				dest = args[1]
			} else {
				dest = contextName + ".dockercontext"
			}
			return runExport(dockerCLI, contextName, dest)
		},
		ValidArgsFunction:     completeContextNames(dockerCLI, 1, true),
		DisableFlagsInUseLine: true,
	}
}

func writeTo(dockerCli command.Cli, reader io.Reader, dest string) error {
	var writer io.Writer
	var printDest bool
	if dest == "-" {
		if dockerCli.Out().IsTerminal() {
			return hint.Wrap(
				errors.New("exported context is a binary tar stream and would corrupt the terminal"),
				"Pass a file path as the second argument, or redirect stdout (e.g. 'docker context export NAME > file.dockercontext').",
			)
		}
		writer = dockerCli.Out()
	} else {
		f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()
		writer = f
		printDest = true
	}
	if _, err := io.Copy(writer, reader); err != nil {
		return err
	}
	if printDest {
		fmt.Fprintf(dockerCli.Err(), "Written file %q\n", dest)
	}
	return nil
}

// runExport exports a Docker context.
func runExport(dockerCLI command.Cli, contextName string, dest string) error {
	if err := store.ValidateContextName(contextName); err != nil && contextName != command.DefaultContextName {
		return err
	}
	reader := store.Export(contextName, dockerCLI.ContextStore())
	defer reader.Close()
	return writeTo(dockerCLI, reader, dest)
}
