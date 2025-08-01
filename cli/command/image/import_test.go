package image

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/image"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNewImportCommandErrors(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		expectedError   string
		imageImportFunc func(source image.ImportSource, ref string, options image.ImportOptions) (io.ReadCloser, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{},
			expectedError: "requires at least 1 argument",
		},
		{
			name:          "import-failed",
			args:          []string{"testdata/import-command-success.input.txt"},
			expectedError: "something went wrong",
			imageImportFunc: func(source image.ImportSource, ref string, options image.ImportOptions) (io.ReadCloser, error) {
				return nil, errors.New("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		cmd := NewImportCommand(test.NewFakeCli(&fakeClient{imageImportFunc: tc.imageImportFunc}))
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(tc.args)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewImportCommandInvalidFile(t *testing.T) {
	cmd := NewImportCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"testdata/import-command-success.unexistent-file"})
	assert.ErrorContains(t, cmd.Execute(), "testdata/import-command-success.unexistent-file")
}

func TestNewImportCommandSuccess(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		imageImportFunc func(source image.ImportSource, ref string, options image.ImportOptions) (io.ReadCloser, error)
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
			imageImportFunc: func(source image.ImportSource, ref string, options image.ImportOptions) (io.ReadCloser, error) {
				assert.Check(t, is.Equal("image:local", ref))
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			name: "message",
			args: []string{"--message", "test message", "-"},
			imageImportFunc: func(source image.ImportSource, ref string, options image.ImportOptions) (io.ReadCloser, error) {
				assert.Check(t, is.Equal("test message", options.Message))
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			name: "change",
			args: []string{"--change", "ENV DEBUG=true", "-"},
			imageImportFunc: func(source image.ImportSource, ref string, options image.ImportOptions) (io.ReadCloser, error) {
				assert.Check(t, is.Equal("ENV DEBUG=true", options.Changes[0]))
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			name: "change legacy syntax",
			args: []string{"--change", "ENV DEBUG true", "-"},
			imageImportFunc: func(source image.ImportSource, ref string, options image.ImportOptions) (io.ReadCloser, error) {
				assert.Check(t, is.Equal("ENV DEBUG true", options.Changes[0]))
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewImportCommand(test.NewFakeCli(&fakeClient{imageImportFunc: tc.imageImportFunc}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.NilError(t, cmd.Execute())
		})
	}
}
