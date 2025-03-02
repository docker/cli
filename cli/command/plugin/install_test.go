package plugin

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/docker/docker/api/types"

	"gotest.tools/v3/assert"
)

func TestInstallErrors(t *testing.T) {
	testCases := []struct {
		description   string
		args          []string
		expectedError string
		installFunc   func(name string, options types.PluginInstallOptions) (io.ReadCloser, error)
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
			args:          []string{"UPPERCASE_REPONAME"},
			expectedError: "invalid",
		},
		{
			description:   "installation error",
			args:          []string{"foo"},
			expectedError: "error installing plugin",
			installFunc: func(name string, options types.PluginInstallOptions) (io.ReadCloser, error) {
				return nil, errors.New("error installing plugin")
			},
		},
		{
			description:   "installation error due to missing image",
			args:          []string{"foo"},
			expectedError: "docker image pull",
			installFunc: func(name string, options types.PluginInstallOptions) (io.ReadCloser, error) {
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

func TestInstallContentTrustErrors(t *testing.T) {
	testCases := []struct {
		description   string
		args          []string
		expectedError string
		notaryFunc    test.NotaryClientFuncType
	}{
		{
			description:   "install plugin, offline notary server",
			args:          []string{"plugin:tag"},
			expectedError: "client is offline",
			notaryFunc:    notary.GetOfflineNotaryRepository,
		},
		{
			description:   "install plugin, uninitialized notary server",
			args:          []string{"plugin:tag"},
			expectedError: "remote trust data does not exist",
			notaryFunc:    notary.GetUninitializedNotaryRepository,
		},
		{
			description:   "install plugin, empty notary server",
			args:          []string{"plugin:tag"},
			expectedError: "No valid trust data for tag",
			notaryFunc:    notary.GetEmptyTargetsNotaryRepository,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				pluginInstallFunc: func(name string, options types.PluginInstallOptions) (io.ReadCloser, error) {
					return nil, errors.New("should not try to install plugin")
				},
			}, test.EnableContentTrust)
			cli.SetNotaryClient(tc.notaryFunc)
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
		installFunc    func(name string, options types.PluginInstallOptions) (io.ReadCloser, error)
	}{
		{
			description:    "install with no additional flags",
			args:           []string{"foo"},
			expectedOutput: "Installed plugin foo\n",
			installFunc: func(name string, options types.PluginInstallOptions) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			description:    "install with disable flag",
			args:           []string{"--disable", "foo"},
			expectedOutput: "Installed plugin foo\n",
			installFunc: func(name string, options types.PluginInstallOptions) (io.ReadCloser, error) {
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
