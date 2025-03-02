//go:build darwin
// +build darwin

package rosetta

import (
	"os"
	"runtime"
	"syscall"
)

// Available returns true if Rosetta is installed/available
func Available() bool {
	_, err := os.Stat("/Library/Apple/usr/share/rosetta")
	return err == nil
}

// Enabled returns true if running in a Rosetta Translated Binary, false otherwise.
func Enabled() bool {
	v, err := syscall.SysctlUint32("sysctl.proc_translated")
	return err == nil && v == 1
}

// NativeArch returns the native architecture, even if binary architecture
// is emulated by Rosetta.
func NativeArch() string {
	if Enabled() && runtime.GOARCH == "amd64" {
		return "arm64"
	}
	return runtime.GOARCH
}
