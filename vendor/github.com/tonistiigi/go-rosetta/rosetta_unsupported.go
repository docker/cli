// +build !darwin

package rosetta

import (
	"runtime"
)

func Enabled() bool {
	return false
}

func NativeArch() string {
	return runtime.GOARCH
}
