package manager

import (
	"os"
	"os/exec"
	"path/filepath"
)

var defaultSystemPluginDirs = []string{
	filepath.Join(os.Getenv("ProgramData"), "Docker", "cli-plugins"),
	filepath.Join(os.Getenv("ProgramFiles"), "Docker", "cli-plugins"),
}

func configureOSSpecificCommand(cmd *exec.Cmd) {
	// no-op
}
