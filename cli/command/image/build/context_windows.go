package build

import (
	"path/filepath"
	"strings"
)

func getContextRoot(srcPath string) (string, error) {
	cr, err := filepath.Abs(srcPath)
	if err != nil {
		return "", err
	}
	return addPrefix(cr), nil
}

// longPathPrefix is the longpath prefix for Windows file paths.
const longPathPrefix = `\\?\`

// addPrefix adds the Windows long path prefix to the path provided if
// it does not already have it.
//
// See https://github.com/moby/moby/pull/15898
//
// This is a copy of [longpath.AddPrefix].
//
// [longpath.AddPrefix]:https://pkg.go.dev/github.com/docker/docker@v28.3.2+incompatible/pkg/longpath#AddPrefix
func addPrefix(path string) string {
	if strings.HasPrefix(path, longPathPrefix) {
		return path
	}
	if strings.HasPrefix(path, `\\`) {
		// This is a UNC path, so we need to add 'UNC' to the path as well.
		return longPathPrefix + `UNC` + path[1:]
	}
	return longPathPrefix + path
}
