package plugin

import (
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/stretchr/testify/assert"
)

func TestCreateErrors(t *testing.T) {

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
			expectedError: "no such file or directory",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		cmd := newCreateCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestCreateErrorOnFileAsContextDir(t *testing.T) {
	tmpFile := fs.NewFile(t, "file-as-context-dir")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	cmd := newCreateCommand(cli)
	cmd.SetArgs([]string{"plugin-foo", tmpFile.Path()})
	cmd.SetOutput(ioutil.Discard)
	testutil.ErrorContains(t, cmd.Execute(), "context must be a directory")
}

func TestCreateErrorOnContextDirWithoutConfig(t *testing.T) {
	tmpDir := fs.NewDir(t, "plugin-create-test")
	defer tmpDir.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	cmd := newCreateCommand(cli)
	cmd.SetArgs([]string{"plugin-foo", tmpDir.Path()})
	cmd.SetOutput(ioutil.Discard)
	testutil.ErrorContains(t, cmd.Execute(), "config.json: no such file or directory")
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
	testutil.ErrorContains(t, cmd.Execute(), "invalid")
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
	testutil.ErrorContains(t, cmd.Execute(), "Error creating plugin")
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
	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "plugin-foo\n", cli.OutBuffer().String())
}
