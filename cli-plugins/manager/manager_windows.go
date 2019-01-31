package manager

import (
	"os"
	"path/filepath"
)

var defaultSystemPluginDirs = []string{
	filepath.Join(os.Getenv("ProgramData"), "Docker", "cli-plugins"),
}
