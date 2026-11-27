//go:build !windows

package manager

// defaultSystemPluginDirs are the platform-specific locations to search
// for plugins in order of preference.
//
// These directories are searched after the directories from the environment
// variable, the configuration file, and the CLI's config-directory.
// See [getPluginDirs] for the complete search order.
var defaultSystemPluginDirs = []string{
	"/usr/local/lib/docker/cli-plugins",
	"/usr/local/libexec/docker/cli-plugins",
	"/usr/lib/docker/cli-plugins",
	"/usr/libexec/docker/cli-plugins",
}
