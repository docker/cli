package cli

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	validContainerNamePattern = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$")
)

// CheckContainerName check container's name is valid or not
func CheckContainerName(name string) error {
	if !validContainerNamePattern.MatchString(strings.TrimPrefix(name, "/")) {
		return fmt.Errorf("container name %s is invalid", name)
	}
	return nil
}

// CheckContainerNames check containers' name is valid or not
func CheckContainerNames(names ...string) error {
	for _, name := range names {
		if err := CheckContainerName(name); err != nil {
			return err
		}
	}
	return nil
}
