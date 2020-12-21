// +build windows
package backends

import (
	"os"
	"path/filepath"
)

func getDockerCliBackendDir() string {
	return filepath.Join(os.Getenv("ProgramData"), "Docker", "cli-backends")
}
