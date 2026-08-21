package connhelper

import (
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
		result := addSSHTimeout(tc.in)
		assert.DeepEqual(t, result, tc.out)
	}
}

func TestConnectionHelperHostIncludesRemoteHost(t *testing.T) {
	testCases := []struct {
		name      string
		daemonURL string
		expected  string
	}{
		{
			name:      "ssh with user and default port",
			daemonURL: "ssh://user@myserver.local",
			expected:  "http://myserver.local",
		},
		{
			name:      "ssh with custom port",
			daemonURL: "ssh://user@myserver.local:2222",
			expected:  "http://myserver.local:2222",
		},
		{
			name:      "ssh with explicit default port",
			daemonURL: "ssh://myserver.local:22",
			expected:  "http://myserver.local",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			helper, err := GetConnectionHelper(tc.daemonURL)
			assert.NilError(t, err)
			assert.Assert(t, helper != nil)
			assert.Equal(t, helper.Host, tc.expected,
				"dummy Host URL must include the remote hostname so connection errors point to the daemon the user configured (docker/cli#5604)")
			assert.Assert(t, helper.Dialer != nil)
		})
	}
}

func TestGetCommandConnectionHelperHost(t *testing.T) {
	helper, err := GetCommandConnectionHelper("fake-command")
	assert.NilError(t, err)
	assert.Assert(t, helper != nil)
	assert.Equal(t, helper.Host, "http://docker.example.com")
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
			assert.DeepEqual(t, result, tc.expected)
		})
	}
}
