package config

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestConfigRemoveErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		configRemoveFunc func(context.Context, string, client.ConfigRemoveOptions) (client.ConfigRemoveResult, error)
		expectedError    string
	}{
		{
			args:          []string{},
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"foo"},
			configRemoveFunc: func(ctx context.Context, name string, options client.ConfigRemoveOptions) (client.ConfigRemoveResult, error) {
				return client.ConfigRemoveResult{}, errors.New("error removing config")
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
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestConfigRemoveWithName(t *testing.T) {
	names := []string{"foo", "bar"}
	var removedConfigs []string
	cli := test.NewFakeCli(&fakeClient{
		configRemoveFunc: func(_ context.Context, name string, _ client.ConfigRemoveOptions) (client.ConfigRemoveResult, error) {
			removedConfigs = append(removedConfigs, name)
			return client.ConfigRemoveResult{}, nil
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
		configRemoveFunc: func(_ context.Context, name string, _ client.ConfigRemoveOptions) (client.ConfigRemoveResult, error) {
			removedConfigs = append(removedConfigs, name)
			if name == "foo" {
				return client.ConfigRemoveResult{}, errors.New("error removing config: " + name)
			}
			return client.ConfigRemoveResult{}, nil
		},
	})

	cmd := newConfigRemoveCommand(cli)
	cmd.SetArgs(names)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.Error(t, cmd.Execute(), "error removing config: foo")
	assert.Check(t, is.DeepEqual(names, removedConfigs))
}
