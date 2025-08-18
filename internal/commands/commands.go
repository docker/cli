package commands

import (
	"os"
	"slices"
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

var (
	commands []func(command.Cli) *cobra.Command
	l        sync.RWMutex
)

// Register pushes the passed in command into an internal queue which can
// be retrieved using the [Commands] function.
func Register(f func(command.Cli) *cobra.Command) {
	l.Lock()
	defer l.Unlock()
	commands = append(commands, f)
}

// RegisterLegacy functions similarly to [Register], but it checks the
// `DOCKER_HIDE_LEGACY_COMMANDS` environment variable and if
// it has been set and is non-empty, the command will be hidden.
func RegisterLegacy(f func(command.Cli) *cobra.Command) {
	l.Lock()
	defer l.Unlock()
	commands = append(commands, func(c command.Cli) *cobra.Command {
		cmd := f(c)
		if os.Getenv("DOCKER_HIDE_LEGACY_COMMANDS") == "" {
			return cmd
		}
		cmdCopy := *cmd
		cmdCopy.Hidden = true
		cmdCopy.Aliases = []string{}
		return &cmdCopy
	})
}

// Commands returns a copy of the internal queue holding registered commands
// added via [Register] or [RegisterLegacy].
func Commands() []func(command.Cli) *cobra.Command {
	l.RLock()
	defer l.RUnlock()
	return slices.Clone(commands)
}
