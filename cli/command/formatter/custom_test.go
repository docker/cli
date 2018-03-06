package formatter

import (
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func compareMultipleValues(t *testing.T, value, expected string) {
	// comma-separated values means probably a map input, which won't
	// be guaranteed to have the same order as our expected value
	// We'll create maps and use reflect.DeepEquals to check instead:
	entriesMap := make(map[string]string)
	expMap := make(map[string]string)
	entries := strings.Split(value, ",")
	expectedEntries := strings.Split(expected, ",")
	for _, entry := range entries {
		keyval := strings.Split(entry, "=")
		entriesMap[keyval[0]] = keyval[1]
	}
	for _, expected := range expectedEntries {
		keyval := strings.Split(expected, "=")
		expMap[keyval[0]] = keyval[1]
	}
	assert.Check(t, is.DeepEqual(expMap, entriesMap))
}
