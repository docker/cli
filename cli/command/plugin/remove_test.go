package plugin

import (
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRemoveErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		pluginRemoveFunc func(name string, options client.PluginRemoveOptions) (client.PluginRemoveResult, error)
		expectedError    string
	}{
		{
			args:          []string{},
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"plugin-foo"},
			pluginRemoveFunc: func(name string, options client.PluginRemoveOptions) (client.PluginRemoveResult, error) {
				return client.PluginRemoveResult{}, errors.New("error removing plugin")
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
	cli := test.NewFakeCli(&fakeClient{})
	cmd := newRemoveCommand(cli)
	cmd.SetArgs([]string{"plugin-foo"})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("plugin-foo\n", cli.OutBuffer().String()))
}

func TestRemoveWithForceOption(t *testing.T) {
	force := false
	cli := test.NewFakeCli(&fakeClient{
		pluginRemoveFunc: func(name string, options client.PluginRemoveOptions) (client.PluginRemoveResult, error) {
			force = options.Force
			return client.PluginRemoveResult{}, nil
		},
	})
	cmd := newRemoveCommand(cli)
	cmd.SetArgs([]string{"plugin-foo"})
	assert.NilError(t, cmd.Flags().Set("force", "true"))
	assert.NilError(t, cmd.Execute())
	assert.Check(t, force)
	assert.Check(t, is.Equal("plugin-foo\n", cli.OutBuffer().String()))
}
