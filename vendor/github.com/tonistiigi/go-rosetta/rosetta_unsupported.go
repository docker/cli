//go:build !darwin
// +build !darwin

package rosetta

import (
	"runtime"
)

// Available returns true if Rosetta is installed/available
func Available() bool {
	return false
}

// Enabled returns true if running in a Rosetta Translated Binary, false otherwise.
func Enabled() bool {
	return false
}

// NativeArch returns the native architecture, even if binary architecture
// is emulated by Rosetta.
func NativeArch() string {
	return runtime.GOARCH
}
