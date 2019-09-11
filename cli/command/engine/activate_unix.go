// +build !windows

package engine

import (
	"golang.org/x/sys/unix"
)

var (
	isRoot = func() bool {
		return unix.Geteuid() == 0
	}
)
