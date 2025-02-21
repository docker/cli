package build // import "docker.com/cli/v28/cli/command/image/build"

import (
	"path/filepath"

	"github.com/docker/docker/pkg/longpath"
)

func getContextRoot(srcPath string) (string, error) {
	cr, err := filepath.Abs(srcPath)
	if err != nil {
		return "", err
	}
	return longpath.AddPrefix(cr), nil
}
