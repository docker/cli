package context

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewContextCommand returns the context cli subcommand
func NewContextCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage contexts",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		newCreateCommand(dockerCli),
		newListCommand(dockerCli),
		newUseCommand(dockerCli),
		newExportCommand(dockerCli),
		newImportCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newUpdateCommand(dockerCli),
		newInspectCommand(dockerCli),
	)
	return cmd
}

const restrictedNamePattern = "^[a-zA-Z0-9][a-zA-Z0-9_.+-]+$"

var restrictedNameRegEx = regexp.MustCompile(restrictedNamePattern)

func validateContextName(name string) error {
	if name == "" {
		return errors.New("context name cannot be empty")
	}
	if name == "default" {
		return errors.New(`"default" is a reserved context name`)
	}
	if !restrictedNameRegEx.MatchString(name) {
		return fmt.Errorf("context name %q is invalid, names are validated against regexp %q", name, restrictedNamePattern)
	}
	return nil
}
