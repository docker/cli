package manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/spf13/cobra"
)

// errPluginNotFound is the error returned when a plugin could not be found.
type errPluginNotFound string

func (e errPluginNotFound) NotFound() {}

func (e errPluginNotFound) Error() string {
	return "Error: No such CLI plugin: " + string(e)
}

type notFound interface{ NotFound() }

// IsNotFound is true if the given error is due to a plugin not being found.
func IsNotFound(err error) bool {
	_, ok := err.(notFound)
	return ok
}

var defaultUserPluginDir = config.Path("cli-plugins")

func getPluginDirs(dockerCli command.Cli) []string {
	var pluginDirs []string

	if cfg := dockerCli.ConfigFile(); cfg != nil {
		pluginDirs = append(pluginDirs, cfg.CLIPluginsExtraDirs...)
	}
	pluginDirs = append(pluginDirs, defaultUserPluginDir)
	pluginDirs = append(pluginDirs, defaultSystemPluginDirs...)
	return pluginDirs
}

// PluginRunCommand returns an "os/exec".Cmd which when .Run() will execute the named plugin.
// The rootcmd argument is referenced to determine the set of builtin commands in order to detect conficts.
// The error returned satisfies the IsNotFound() predicate if no plugin was found or if the first candidate plugin was invalid somehow.
func PluginRunCommand(dockerCli command.Cli, name string, rootcmd *cobra.Command) (*exec.Cmd, error) {
	// This uses the full original args, not the args which may
	// have been provided by cobra to our caller. This is because
	// they lack e.g. global options which we must propagate here.
	args := os.Args[1:]
	if !pluginNameRe.MatchString(name) {
		// We treat this as "not found" so that callers will
		// fallback to their "invalid" command path.
		return nil, errPluginNotFound(name)
	}
	exename := NamePrefix + name
	if runtime.GOOS == "windows" {
		exename = exename + ".exe"
	}
	for _, d := range getPluginDirs(dockerCli) {
		path := filepath.Join(d, exename)

		// We stat here rather than letting the exec tell us
		// ENOENT because the latter does not distinguish a
		// file not existing from its dynamic loader or one of
		// its libraries not existing.
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		c := &candidate{path: path}
		plugin, err := newPlugin(c, rootcmd)
		if err != nil {
			return nil, err
		}
		if plugin.Err != nil {
			return nil, errPluginNotFound(name)
		}
		cmd := exec.Command(plugin.Path, args...)
		// Using dockerCli.{In,Out,Err}() here results in a hang until something is input.
		// See: - https://github.com/golang/go/issues/10338
		//      - https://github.com/golang/go/commit/d000e8742a173aa0659584aa01b7ba2834ba28ab
		// os.Stdin is a *os.File which avoids this behaviour. We don't need the functionality
		// of the wrappers here anyway.
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd, nil
	}
	return nil, errPluginNotFound(name)
}
