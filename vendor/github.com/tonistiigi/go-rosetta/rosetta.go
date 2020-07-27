// +build darwin

package rosetta

import (
	"runtime"
	"syscall"
)

func Enabled() bool {
	v, err := syscall.SysctlUint32("sysctl.proc_translated")
	return err == nil && v == 1
}

func NativeArch() string {
	if Enabled() && runtime.GOARCH == "amd64" {
		return "arm64"
	}
	return runtime.GOARCH
}
