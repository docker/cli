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

func TestPluginDisableErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		expectedError     string
		pluginDisableFunc func(name string, disableOptions client.PluginDisableOptions) (client.PluginDisableResult, error)
	}{
		{
			args:          []string{},
			expectedError: "requires 1 argument",
		},
		{
			args:          []string{"too", "many", "arguments"},
			expectedError: "requires 1 argument",
		},
		{
			args:          []string{"plugin-foo"},
			expectedError: "error disabling plugin",
			pluginDisableFunc: func(name string, disableOptions client.PluginDisableOptions) (client.PluginDisableResult, error) {
				return client.PluginDisableResult{}, errors.New("error disabling plugin")
			},
		},
	}

	for _, tc := range testCases {
		cmd := newDisableCommand(
			test.NewFakeCli(&fakeClient{
				pluginDisableFunc: tc.pluginDisableFunc,
			}))
		cmd.SetArgs(tc.args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestPluginDisable(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		pluginDisableFunc: func(name string, disableOptions client.PluginDisableOptions) (client.PluginDisableResult, error) {
			return client.PluginDisableResult{}, nil
		},
	})
	cmd := newDisableCommand(cli)
	cmd.SetArgs([]string{"plugin-foo"})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("plugin-foo\n", cli.OutBuffer().String()))
}
