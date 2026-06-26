package container

import (
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestStartValidateFlags(t *testing.T) {
	for _, tc := range []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "with invalid --detach-keys",
			args:        []string{"--detach-keys", "shift-a", "myimage"},
			expectedErr: "invalid detach keys (shift-a):",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newStartCommand(test.NewFakeCli(&fakeClient{}))
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
