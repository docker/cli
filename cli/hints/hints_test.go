package hints

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestEnabled(t *testing.T) {
	tests := []struct {
		doc      string
		env      string
		expected bool
	}{
		{
			doc:      "default",
			expected: true,
		},
		{
			doc:      "DOCKER_CLI_HINTS=1",
			env:      "1",
			expected: true,
		},
		{
			doc:      "DOCKER_CLI_HINTS=true",
			env:      "true",
			expected: true,
		},
		{
			doc:      "DOCKER_CLI_HINTS=0",
			env:      "0",
			expected: false,
		},
		{
			doc:      "DOCKER_CLI_HINTS=false",
			env:      "false",
			expected: false,
		},
		{
			doc:      "DOCKER_CLI_HINTS=not-a-bool",
			env:      "not-a-bool",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			t.Setenv("DOCKER_CLI_HINTS", tc.env)
			assert.Equal(t, Enabled(), tc.expected)
		})
	}
}
