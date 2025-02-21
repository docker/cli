package plugin // import "docker.com/cli/v28/cli/command/plugin"

import (
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/v28/internal/test"
	"github.com/docker/docker/api/types"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRemoveErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		pluginRemoveFunc func(name string, options types.PluginRemoveOptions) error
		expectedError    string
	}{
		{
			args:          []string{},
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"plugin-foo"},
			pluginRemoveFunc: func(name string, options types.PluginRemoveOptions) error {
				return errors.New("error removing plugin")
			},
			expectedError: "error removing plugin",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			pluginRemoveFunc: tc.pluginRemoveFunc,
		})
		cmd := newRemoveCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestRemove(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		pluginRemoveFunc: func(name string, options types.PluginRemoveOptions) error {
			return nil
		},
	})
	cmd := newRemoveCommand(cli)
	cmd.SetArgs([]string{"plugin-foo"})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("plugin-foo\n", cli.OutBuffer().String()))
}

func TestRemoveWithForceOption(t *testing.T) {
	force := false
	cli := test.NewFakeCli(&fakeClient{
		pluginRemoveFunc: func(name string, options types.PluginRemoveOptions) error {
			force = options.Force
			return nil
		},
	})
	cmd := newRemoveCommand(cli)
	cmd.SetArgs([]string{"plugin-foo"})
	cmd.Flags().Set("force", "true")
	assert.NilError(t, cmd.Execute())
	assert.Check(t, force)
	assert.Check(t, is.Equal("plugin-foo\n", cli.OutBuffer().String()))
}
