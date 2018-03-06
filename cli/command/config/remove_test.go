package config

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func TestConfigRemoveErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		configRemoveFunc func(string) error
		expectedError    string
	}{
		{
			args:          []string{},
			expectedError: "requires at least 1 argument.",
		},
		{
			args: []string{"foo"},
			configRemoveFunc: func(name string) error {
				return errors.Errorf("error removing config")
			},
			expectedError: "error removing config",
		},
	}
	for _, tc := range testCases {
		cmd := newConfigRemoveCommand(
			test.NewFakeCli(&fakeClient{
				configRemoveFunc: tc.configRemoveFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestConfigRemoveWithName(t *testing.T) {
	names := []string{"foo", "bar"}
	var removedConfigs []string
	cli := test.NewFakeCli(&fakeClient{
		configRemoveFunc: func(name string) error {
			removedConfigs = append(removedConfigs, name)
			return nil
		},
	})
	cmd := newConfigRemoveCommand(cli)
	cmd.SetArgs(names)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.DeepEqual(names, strings.Split(strings.TrimSpace(cli.OutBuffer().String()), "\n")))
	assert.Check(t, is.DeepEqual(names, removedConfigs))
}

func TestConfigRemoveContinueAfterError(t *testing.T) {
	names := []string{"foo", "bar"}
	var removedConfigs []string

	cli := test.NewFakeCli(&fakeClient{
		configRemoveFunc: func(name string) error {
			removedConfigs = append(removedConfigs, name)
			if name == "foo" {
				return errors.Errorf("error removing config: %s", name)
			}
			return nil
		},
	})

	cmd := newConfigRemoveCommand(cli)
	cmd.SetArgs(names)
	cmd.SetOutput(ioutil.Discard)
	assert.Error(t, cmd.Execute(), "error removing config: foo")
	assert.Check(t, is.DeepEqual(names, removedConfigs))
}
