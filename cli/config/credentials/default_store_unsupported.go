//go:build !windows && !darwin && !linux

package credentials

const (
	preferredHelper = ""
	defaultHelper   = ""
)
