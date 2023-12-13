package manager

import (
	"fmt"
	"path/filepath"
	"strings"
)

// This is made slightly more complex due to needing to be case insensitive.
func trimExeSuffix(s string) (string, error) {
	ext := filepath.Ext(s)
	if ext == "" {
		return "", fmt.Errorf("path %q lacks required file extension", s)
	}

	exe := ".exe"
	if !strings.EqualFold(ext, exe) {
		return "", fmt.Errorf("path %q lacks required %q suffix", s, exe)
	}
	return strings.TrimSuffix(s, ext), nil
}

func addExeSuffix(s string) string {
	return s + ".exe"
}
