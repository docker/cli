package plugin

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"

	"gotest.tools/v3/assert"
)

func TestInstallErrors(t *testing.T) {
	testCases := []struct {
		description   string
		args          []string
		expectedError string
		installFunc   func(name string, options client.PluginInstallOptions) (io.ReadCloser, error)
	}{
		{
			description:   "insufficient number of arguments",
			args:          []string{},
			expectedError: "requires at least 1 argument",
		},
		{
			description:   "invalid alias",
			args:          []string{"foo", "--alias", "UPPERCASE_ALIAS"},
			expectedError: "invalid",
		},
		{
			description:   "invalid plugin name",
			args:          []string{"UPPERCASE_REPO_NAME"},
			expectedError: "invalid",
		},
		{
			description:   "installation error",
			args:          []string{"foo"},
			expectedError: "error installing plugin",
			installFunc: func(name string, options client.PluginInstallOptions) (io.ReadCloser, error) {
				return nil, errors.New("error installing plugin")
			},
		},
		{
			description:   "installation error due to missing image",
			args:          []string{"foo"},
			expectedError: "docker image pull",
			installFunc: func(name string, options client.PluginInstallOptions) (io.ReadCloser, error) {
				return nil, errors.New("(image) when fetching")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{pluginInstallFunc: tc.installFunc})
			cmd := newInstallCommand(cli)
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestInstall(t *testing.T) {
	testCases := []struct {
		description    string
		args           []string
		expectedOutput string
		installFunc    func(name string, options client.PluginInstallOptions) (io.ReadCloser, error)
	}{
		{
			description:    "install with no additional flags",
			args:           []string{"foo"},
			expectedOutput: "Installed plugin foo\n",
			installFunc: func(name string, options client.PluginInstallOptions) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			description:    "install with disable flag",
			args:           []string{"--disable", "foo"},
			expectedOutput: "Installed plugin foo\n",
			installFunc: func(name string, options client.PluginInstallOptions) (io.ReadCloser, error) {
				assert.Check(t, options.Disabled)
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{pluginInstallFunc: tc.installFunc})
			cmd := newInstallCommand(cli)
			cmd.SetArgs(tc.args)
			assert.NilError(t, cmd.Execute())
			assert.Check(t, strings.Contains(cli.OutBuffer().String(), tc.expectedOutput))
		})
	}
}
