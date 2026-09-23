//go:build !windows

package container

import (
	"os"
	"strconv"
	"syscall"
)

// addSocketGroup appends the GID of the socket file at path to groupAdd, so
// non-root users can access the socket without an explicit --group-add flag.
// Errors are silently ignored; this is best-effort.
func addSocketGroup(groupAdd *[]string, path string) {
	fi, err := os.Stat(path)
	if err != nil {
		return
	}
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	*groupAdd = append(*groupAdd, strconv.FormatUint(uint64(stat.Gid), 10))
}
