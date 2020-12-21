// +build !windows

package backends

func getDockerCliBackendDir() string {
	// TODO check , "/usr/local/libexec/docker/cli-backends", if "/usr/local/lib/" does not exist
	return "/usr/local/lib/docker/cli-backends"
}
