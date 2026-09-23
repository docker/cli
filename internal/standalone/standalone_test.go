package standalone

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestEnabled(t *testing.T) {
	tests := []struct {
		value    string
		unset    bool
		expected bool
	}{
		{unset: true, expected: false},
		{value: "", expected: false},
		{value: "0", expected: false},
		{value: "false", expected: false},
		{value: "1", expected: true},
		{value: "true", expected: true},
		{value: "yes", expected: true},
	}
	for _, tc := range tests {
		name := tc.value
		if tc.unset {
			name = "unset"
		}
		t.Run(name, func(t *testing.T) {
			if tc.unset {
				t.Setenv(EnvEnabled, "")
				assert.NilError(t, os.Unsetenv(EnvEnabled))
			} else {
				t.Setenv(EnvEnabled, tc.value)
			}
			assert.Check(t, is.Equal(Enabled(), tc.expected))
		})
	}
}

func TestIsLoggerInvocation(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{name: "logger", args: []string{"docker", loggerArg, "/run/dir"}, expected: true},
		{name: "no args", args: []string{"docker"}},
		{name: "missing dir", args: []string{"docker", loggerArg}},
		{name: "other command", args: []string{"docker", "run", "alpine"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Check(t, is.Equal(IsLoggerInvocation(tc.args), tc.expected))
		})
	}
}
