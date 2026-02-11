// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package test

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// CompareMultipleValues compares comma-separated values, whatever the order is
func CompareMultipleValues(t *testing.T, value, expected string) {
	t.Helper()
	// comma-separated values means probably a map input, which won't
	// be guaranteed to have the same order as our expected value
	// We'll create maps and use reflect.DeepEquals to check instead:
	entriesMap := make(map[string]string)
	for entry := range strings.SplitSeq(value, ",") {
		k, v, _ := strings.Cut(entry, "=")
		entriesMap[k] = v
	}
	expMap := make(map[string]string)
	for exp := range strings.SplitSeq(expected, ",") {
		k, v, _ := strings.Cut(exp, "=")
		expMap[k] = v
	}
	assert.Check(t, is.DeepEqual(expMap, entriesMap))
}
