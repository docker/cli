package formatter

import (
	"testing"

	"github.com/docker/cli/internal/test"
)

// Deprecated: use internal/test.CompareMultipleValues instead
func compareMultipleValues(t *testing.T, value, expected string) {
	test.CompareMultipleValues(t, value, expected)
}
