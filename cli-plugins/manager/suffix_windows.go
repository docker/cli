package manager

import (
	"strings"

	"github.com/pkg/errors"
)

func trimExeSuffix(s string) (string, error) {
	exe := ".exe"
	if !strings.HasSuffix(s, exe) {
		return "", errors.Errorf("lacks required %q suffix", exe)
	}
	return strings.TrimSuffix(s, exe), nil
}

func addExeSuffix(s string) string {
	return s + ".exe"
}
