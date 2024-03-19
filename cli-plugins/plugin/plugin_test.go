package plugin

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestUserAgent(t *testing.T) {
	tcs := []struct {
		expected string
		name     string
		version  string
	}{
		{
			expected: "docker-cli-plugin-whalesay/0.0.1",
			name:     "whalesay",
			version:  "0.0.1",
		},
		{
			expected: "docker-cli-plugin-whalesay/unknown",
			name:     "whalesay",
		},
		{
			expected: "docker-cli-plugin-unknown/unknown",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.expected, func(t *testing.T) {
			ua := pluginUserAgent(tc.name, tc.version)
			assert.Equal(t, ua, tc.expected)
		})
	}
}
