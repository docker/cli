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
