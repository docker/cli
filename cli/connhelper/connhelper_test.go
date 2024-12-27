package connhelper

import (
	"reflect"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSSHFlags(t *testing.T) {
	testCases := []struct {
		in  []string
		out []string
	}{
		{
			in:  []string{},
			out: []string{"-o ConnectTimeout=30"},
		},
		{
			in:  []string{"option", "-o anotherOption"},
			out: []string{"option", "-o anotherOption", "-o ConnectTimeout=30"},
		},
		{
			in:  []string{"-o ConnectTimeout=5", "anotherOption"},
			out: []string{"-o ConnectTimeout=5", "anotherOption"},
		},
	}

	for _, tc := range testCases {
		assert.DeepEqual(t, addSSHTimeout(tc.in), tc.out)
	}
}

func TestDisablePseudoTerminalAllocation(t *testing.T) {
	testCases := []struct {
		name     string
		sshFlags []string
		expected []string
	}{
		{
			name:     "No -T flag present",
			sshFlags: []string{"-v", "-oStrictHostKeyChecking=no"},
			expected: []string{"-v", "-oStrictHostKeyChecking=no", "-T"},
		},
		{
			name:     "Already contains -T flag",
			sshFlags: []string{"-v", "-T", "-oStrictHostKeyChecking=no"},
			expected: []string{"-v", "-T", "-oStrictHostKeyChecking=no"},
		},
		{
			name:     "Empty sshFlags",
			sshFlags: []string{},
			expected: []string{"-T"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := disablePseudoTerminalAllocation(tc.sshFlags)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestDockerSSHBinaryOverride(t *testing.T) {
	testCases := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default",
			env:      "",
			expected: "docker",
		},
		{
			name:     "Override",
			env:      "other-binary",
			expected: "other-binary",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(DockerSSHRemoteBinaryEnv, tc.env)
			result := dockerSSHRemoteBinary()
			assert.Check(t, is.Equal(result, tc.expected))
		})
	}
}
