//go:build !windows

package manager // import "docker.com/cli/v28/cli-plugins/manager"

func trimExeSuffix(s string) (string, error) {
	return s, nil
}

func addExeSuffix(s string) string {
	return s
}
