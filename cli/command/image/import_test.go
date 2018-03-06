package image

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func TestNewImportCommandErrors(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		expectedError   string
		imageImportFunc func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{},
			expectedError: "requires at least 1 argument.",
		},
		{
			name:          "import-failed",
			args:          []string{"testdata/import-command-success.input.txt"},
			expectedError: "something went wrong",
			imageImportFunc: func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
				return nil, errors.Errorf("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		cmd := NewImportCommand(test.NewFakeCli(&fakeClient{imageImportFunc: tc.imageImportFunc}))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewImportCommandInvalidFile(t *testing.T) {
	cmd := NewImportCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"testdata/import-command-success.unexistent-file"})
	assert.ErrorContains(t, cmd.Execute(), "testdata/import-command-success.unexistent-file")
}

func TestNewImportCommandSuccess(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		imageImportFunc func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error)
	}{
		{
			name: "simple",
			args: []string{"testdata/import-command-success.input.txt"},
		},
		{
			name: "terminal-source",
			args: []string{"-"},
		},
		{
			name: "double",
			args: []string{"-", "image:local"},
			imageImportFunc: func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
				assert.Check(t, is.Equal("image:local", ref))
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			name: "message",
			args: []string{"--message", "test message", "-"},
			imageImportFunc: func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
				assert.Check(t, is.Equal("test message", options.Message))
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			name: "change",
			args: []string{"--change", "ENV DEBUG true", "-"},
			imageImportFunc: func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
				assert.Check(t, is.Equal("ENV DEBUG true", options.Changes[0]))
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		},
	}
	for _, tc := range testCases {
		cmd := NewImportCommand(test.NewFakeCli(&fakeClient{imageImportFunc: tc.imageImportFunc}))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		assert.NilError(t, cmd.Execute())
	}
}
