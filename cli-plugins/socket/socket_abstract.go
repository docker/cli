//go:build windows || linux

package socket

func socketName(basename string) string {
	// Address of an abstract socket -- this socket can be opened by name,
	// but is not present in the filesystem.
	return "@" + basename
}
