//go:build windows || linux

package socket // import "docker.com/cli/v28/cli-plugins/socket"

func socketName(basename string) string {
	// Address of an abstract socket -- this socket can be opened by name,
	// but is not present in the filesystem.
	return "@" + basename
}
