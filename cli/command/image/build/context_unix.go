//go:build !windows

package build // import "docker.com/cli/v28/cli/command/image/build"

import (
	"path/filepath"
)

func getContextRoot(srcPath string) (string, error) {
	return filepath.Join(srcPath, "."), nil
}
