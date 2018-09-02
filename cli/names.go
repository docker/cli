package cli

import (
	"strings"

	"github.com/docker/cli/cli/names"
)

var (
	validContainerNamePattern = names.RestrictedNamePattern
)

// CheckContainerName check container's name is valid or not
func CheckContainerName(name string) bool {
	if len(name) == 0 {
		return false
	}
	return validContainerNamePattern.MatchString(strings.TrimPrefix(name, "/"))
}
