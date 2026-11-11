package manager

import (
	"os"
	"path/filepath"
)

// defaultSystemPluginDirs are the platform-specific locations to search
// for plugins in order of preference.
//
// These directories are searched after the directories from the environment
// variable, the configuration file, and the CLI's config-directory.
// See [getPluginDirs] for the complete search order.
var defaultSystemPluginDirs = []string{
	filepath.Join(os.Getenv("ProgramFiles"), "Docker", "cli-plugins"),
}
