package commands

import (
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

var commands []func(command.Cli) *cobra.Command

// Register pushes the passed in command into an internal queue which can
// be retrieved using the [Commands] function. It is designed to be called
// in an init function and is not safe for concurrent use.
func Register(f func(command.Cli) *cobra.Command) {
	commands = append(commands, f)
}

// RegisterLegacy functions similarly to [Register], but it checks the
// "DOCKER_HIDE_LEGACY_COMMANDS" environment variable and if it has been
// set and is non-empty, the command will be hidden. It is designed to be called
// in an init function and is not safe for concurrent use.
func RegisterLegacy(f func(command.Cli) *cobra.Command) {
	commands = append(commands, func(c command.Cli) *cobra.Command {
		if os.Getenv("DOCKER_HIDE_LEGACY_COMMANDS") == "" {
			return f(c)
		}
		cmd := f(c)
		cmd.Hidden = true
		cmd.Aliases = []string{}
		return cmd
	})
}

// Commands returns the internal queue holding registered commands added
// via [Register] and [RegisterLegacy].
func Commands() []func(command.Cli) *cobra.Command {
	return commands
}
