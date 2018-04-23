package output

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
)

// Assert checks wether the output contains the specified lines
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

// Prefix returns whether if the line has the specified string as prefix
func Prefix(expected string) func(string) error {
	return func(actual string) error {
		if strings.HasPrefix(actual, expected) {
			return nil
		}
		return errors.Errorf("expected %s to start with %s", actual, expected)
	}
}

// Equals returns wether the line is the same as the specified string
func Equals(expected string) func(string) error {
	return func(actual string) error {
		if expected == actual {
			return nil
		}
		return errors.Errorf("got %s, expected %s", actual, expected)
	}
}
