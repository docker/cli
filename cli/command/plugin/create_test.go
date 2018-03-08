package plugin

import (
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/fs"
)

func TestCreateErrors(t *testing.T) {
	noSuchFile := "no such file or directory"
	if runtime.GOOS == "windows" {
		noSuchFile = "The system cannot find the file specified."
	}
	testCases := []struct {
		args          []string
		expectedError string
	}{
		{
			args:          []string{},
			expectedError: "requires at least 2 arguments",
		},
		{
			args:          []string{"INVALID_TAG", "context-dir"},
			expectedError: "invalid",
		},
		{
			args:          []string{"plugin-foo", "nonexistent_context_dir"},
			expectedError: noSuchFile,
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		cmd := newCreateCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCreateErrorOnFileAsContextDir(t *testing.T) {
	tmpFile := fs.NewFile(t, "file-as-context-dir")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	cmd := newCreateCommand(cli)
	cmd.SetArgs([]string{"plugin-foo", tmpFile.Path()})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "context must be a directory")
}

func TestCreateErrorOnContextDirWithoutConfig(t *testing.T) {
	tmpDir := fs.NewDir(t, "plugin-create-test")
	defer tmpDir.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	cmd := newCreateCommand(cli)
	cmd.SetArgs([]string{"plugin-foo", tmpDir.Path()})
	cmd.SetOutput(ioutil.Discard)

	expectedErr := "config.json: no such file or directory"
	if runtime.GOOS == "windows" {
		expectedErr = "config.json: The system cannot find the file specified."
	}
	assert.ErrorContains(t, cmd.Execute(), expectedErr)
}

func TestCreateErrorOnInvalidConfig(t *testing.T) {
	tmpDir := fs.NewDir(t, "plugin-create-test",
		fs.WithDir("rootfs"),
		fs.WithFile("config.json", "invalid-config-contents"))
	defer tmpDir.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	cmd := newCreateCommand(cli)
	cmd.SetArgs([]string{"plugin-foo", tmpDir.Path()})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "invalid")
}

func TestCreateErrorFromDaemon(t *testing.T) {
	tmpDir := fs.NewDir(t, "plugin-create-test",
		fs.WithDir("rootfs"),
		fs.WithFile("config.json", `{ "Name": "plugin-foo" }`))
	defer tmpDir.Remove()

	cli := test.NewFakeCli(&fakeClient{
		pluginCreateFunc: func(createContext io.Reader, createOptions types.PluginCreateOptions) error {
			return fmt.Errorf("Error creating plugin")
		},
	})

	cmd := newCreateCommand(cli)
	cmd.SetArgs([]string{"plugin-foo", tmpDir.Path()})
	cmd.SetOutput(ioutil.Discard)
	assert.ErrorContains(t, cmd.Execute(), "Error creating plugin")
}

func TestCreatePlugin(t *testing.T) {
	tmpDir := fs.NewDir(t, "plugin-create-test",
		fs.WithDir("rootfs"),
		fs.WithFile("config.json", `{ "Name": "plugin-foo" }`))
	defer tmpDir.Remove()

	cli := test.NewFakeCli(&fakeClient{
		pluginCreateFunc: func(createContext io.Reader, createOptions types.PluginCreateOptions) error {
			return nil
		},
	})

	cmd := newCreateCommand(cli)
	cmd.SetArgs([]string{"plugin-foo", tmpDir.Path()})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal("plugin-foo\n", cli.OutBuffer().String()))
}
