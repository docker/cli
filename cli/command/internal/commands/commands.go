package commands

import (
	"os"
	"slices"
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/spf13/cobra"
)

var (
	commands []func(command.Cli) *cobra.Command
	l        sync.RWMutex
)

func RegisterCommand(f func(cli.Cli) *cobra.Command) {
	l.Lock()
	defer l.Unlock()
	commands = append(commands, f)
}

// MaybeHideLegacy checks the `DOCKER_HIDE_LEGACY_COMMANDS` environment variable and if
// it has been set and is non-empty, the legacy command will be hidden.
// Legacy commands are `docker ps`, `docker exec`, etc)
func MaybeHideLegacy(f func(command.Cli) *cobra.Command) func(command.Cli) *cobra.Command {
	return func(c command.Cli) *cobra.Command {
		cmd := f(c)
		// If the environment variable with name "DOCKER_HIDE_LEGACY_COMMANDS" is not empty,
		// these legacy commands (such as `docker ps`, `docker exec`, etc)
		// will not be shown in output console.
		if os.Getenv("DOCKER_HIDE_LEGACY_COMMANDS") == "" {
			return cmd
		}
		cmdCopy := *cmd
		cmdCopy.Hidden = true
		cmdCopy.Aliases = []string{}
		return &cmdCopy
	}
}

func Commands() []func(command.Cli) *cobra.Command {
	l.RLock()
	defer l.RUnlock()
	return slices.Clone(commands)
}
