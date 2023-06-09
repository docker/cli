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
		assert.DeepEqual(t, addSSHTimeout(tc.in), tc.out)
	}
}
