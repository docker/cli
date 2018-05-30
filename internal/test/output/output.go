package output

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
)

// Assert checks output lines at specified locations
func Assert(t *testing.T, actual string, expectedLines map[int]func(string) error) {
	t.Helper()
	for i, line := range strings.Split(actual, "\n") {
		cmp, ok := expectedLines[i]
		if !ok {
			continue
		}
		if err := cmp(line); err != nil {
			t.Errorf("line %d: %s", i, err)
		}
	}
	if t.Failed() {
		t.Log(actual)
	}
}

// Prefix returns whether the line has the specified string as prefix
func Prefix(expected string) func(string) error {
	return func(actual string) error {
		if strings.HasPrefix(actual, expected) {
			return nil
		}
		return errors.Errorf("expected %q to start with %q", actual, expected)
	}
}

// Suffix returns whether the line has the specified string as suffix
func Suffix(expected string) func(string) error {
	return func(actual string) error {
		if strings.HasSuffix(actual, expected) {
			return nil
		}
		return errors.Errorf("expected %q to end with %q", actual, expected)
	}
}

// Contains returns whether the line contains the specified string
func Contains(expected string) func(string) error {
	return func(actual string) error {
		if strings.Contains(actual, expected) {
			return nil
		}
		return errors.Errorf("expected %q to contain %q", actual, expected)
	}
}

// Equals returns wether the line is the same as the specified string
func Equals(expected string) func(string) error {
	return func(actual string) error {
		if expected == actual {
			return nil
		}
		return errors.Errorf("got %q, expected %q", actual, expected)
	}
}
