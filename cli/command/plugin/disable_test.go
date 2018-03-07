package plugin

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestPluginDisableErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		expectedError     string
		pluginDisableFunc func(name string, disableOptions types.PluginDisableOptions) error
	}{
		{
			args:          []string{},
			expectedError: "requires exactly 1 argument",
		},
		{
			args:          []string{"too", "many", "arguments"},
			expectedError: "requires exactly 1 argument",
		},
		{
			args:          []string{"plugin-foo"},
			expectedError: "Error disabling plugin",
			pluginDisableFunc: func(name string, disableOptions types.PluginDisableOptions) error {
				return fmt.Errorf("Error disabling plugin")
			},
		},
	}

	for _, tc := range testCases {
		cmd := newDisableCommand(
			test.NewFakeCli(&fakeClient{
				pluginDisableFunc: tc.pluginDisableFunc,
			}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestPluginDisable(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		pluginDisableFunc: func(name string, disableOptions types.PluginDisableOptions) error {
			return nil
		},
	})
	cmd := newDisableCommand(cli)
	cmd.SetArgs([]string{"plugin-foo"})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("plugin-foo\n", cli.OutBuffer().String()))
}
