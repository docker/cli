package system

import (
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestInspectValidateFlagsAndArgs(t *testing.T) {
	for _, tc := range []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "empty type",
			args:        []string{"--type", "", "something"},
			expectedErr: `type is empty: must be one of "config", "container", "image", "network", "node", "plugin", "secret", "service", "task", "volume"`,
		},
		{
			name:        "unknown type",
			args:        []string{"--type", "unknown", "something"},
			expectedErr: `unknown type: "unknown": must be one of "config", "container", "image", "network", "node", "plugin", "secret", "service", "task", "volume"`,
		},
		{
			name:        "no arg",
			args:        []string{},
			expectedErr: `inspect: 'inspect' requires at least 1 argument`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newInspectCommand(test.NewFakeCli(&fakeClient{}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			} else {
				assert.Check(t, is.Nil(err))
			}
		})
	}
}
