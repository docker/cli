package manager

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetNaiveFlags(t *testing.T) {
	testCases := []struct {
		args          []string
		expectedFlags map[string]string
	}{
		{
			args:          []string{"docker"},
			expectedFlags: map[string]string{},
		},
		{
			args: []string{"docker", "build", "-q", "--file", "test.Dockerfile", "."},
			expectedFlags: map[string]string{
				"q":    "",
				"file": "",
			},
		},
		{
			args: []string{"docker", "--context", "a-context", "pull", "-q", "--progress", "auto", "alpine"},
			expectedFlags: map[string]string{
				"context":  "",
				"q":        "",
				"progress": "",
			},
		},
	}

	for _, tc := range testCases {
		assert.DeepEqual(t, getNaiveFlags(tc.args), tc.expectedFlags)
	}
}

func TestPluginMatch(t *testing.T) {
	testCases := []struct {
		commandString string
		pluginConfig  map[string]string
		expectedMatch string
		expectedOk    bool
	}{
		{
			commandString: "image ls",
			pluginConfig: map[string]string{
				"hooks": "image",
			},
			expectedMatch: "image",
			expectedOk:    true,
		},
		{
			commandString: "context ls",
			pluginConfig: map[string]string{
				"hooks": "build",
			},
			expectedMatch: "",
			expectedOk:    false,
		},
		{
			commandString: "context ls",
			pluginConfig: map[string]string{
				"hooks": "context ls",
			},
			expectedMatch: "context ls",
			expectedOk:    true,
		},
		{
			commandString: "image ls",
			pluginConfig: map[string]string{
				"hooks": "image ls,image",
			},
			expectedMatch: "image ls",
			expectedOk:    true,
		},
		{
			commandString: "image ls",
			pluginConfig: map[string]string{
				"hooks": "",
			},
			expectedMatch: "",
			expectedOk:    false,
		},
		{
			commandString: "image inspect",
			pluginConfig: map[string]string{
				"hooks": "image i",
			},
			expectedMatch: "",
			expectedOk:    false,
		},
		{
			commandString: "image inspect",
			pluginConfig: map[string]string{
				"hooks": "image",
			},
			expectedMatch: "image",
			expectedOk:    true,
		},
	}

	for _, tc := range testCases {
		match, ok := pluginMatch(tc.pluginConfig, tc.commandString)
		assert.Equal(t, ok, tc.expectedOk)
		assert.Equal(t, match, tc.expectedMatch)
	}
}
