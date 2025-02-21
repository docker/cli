//go:build !windows && !darwin && !linux

package credentials // import "docker.com/cli/v28/cli/config/credentials"

func defaultCredentialsStore() string {
	return ""
}
