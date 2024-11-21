package image

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/image"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNewSaveCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		isTerminal    bool
		expectedError string
		imageSaveFunc func(images []string, options image.SaveOptions) (io.ReadCloser, error)
	}{
		{
			name:          "wrong args",
			args:          []string{},
			expectedError: "requires at least 1 argument",
		},
		{
			name:          "output to terminal",
			args:          []string{"output", "file", "arg1"},
			isTerminal:    true,
			expectedError: "cowardly refusing to save to a terminal. Use the -o flag or redirect",
		},
		{
			name:          "ImageSave fail",
			args:          []string{"arg1"},
			isTerminal:    false,
			expectedError: "error saving image",
			imageSaveFunc: func(images []string, options image.SaveOptions) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("")), errors.Errorf("error saving image")
			},
		},
		{
			name:          "output directory does not exist",
			args:          []string{"-o", "fakedir/out.tar", "arg1"},
			expectedError: "failed to save image: invalid output path: directory \"fakedir\" does not exist",
		},
		{
			name:          "output file is irregular",
			args:          []string{"-o", "/dev/null", "arg1"},
			expectedError: "failed to save image: invalid output path: \"/dev/null\" must be a directory or a regular file",
		},
		{
			name:          "invalid platform",
			args:          []string{"--platform", "<invalid>", "arg1"},
			expectedError: `invalid platform`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageSaveFunc: tc.imageSaveFunc})
			cli.Out().SetIsTerminal(tc.isTerminal)
			cmd := NewSaveCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewSaveCommandSuccess(t *testing.T) {
	testCases := []struct {
		args          []string
		isTerminal    bool
		imageSaveFunc func(images []string, options image.SaveOptions) (io.ReadCloser, error)
		deferredFunc  func()
	}{
		{
			args:       []string{"-o", "save_tmp_file", "arg1"},
			isTerminal: true,
			imageSaveFunc: func(images []string, _ image.SaveOptions) (io.ReadCloser, error) {
				assert.Assert(t, is.Len(images, 1))
				assert.Check(t, is.Equal("arg1", images[0]))
				return io.NopCloser(strings.NewReader("")), nil
			},
			deferredFunc: func() {
				_ = os.Remove("save_tmp_file")
			},
		},
		{
			args:       []string{"arg1", "arg2"},
			isTerminal: false,
			imageSaveFunc: func(images []string, _ image.SaveOptions) (io.ReadCloser, error) {
				assert.Assert(t, is.Len(images, 2))
				assert.Check(t, is.Equal("arg1", images[0]))
				assert.Check(t, is.Equal("arg2", images[1]))
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			args:       []string{"--platform", "linux/amd64", "arg1"},
			isTerminal: false,
			imageSaveFunc: func(images []string, options image.SaveOptions) (io.ReadCloser, error) {
				assert.Assert(t, is.Len(images, 1))
				assert.Check(t, is.Equal("arg1", images[0]))
				assert.Check(t, is.DeepEqual([]ocispec.Platform{{OS: "linux", Architecture: "amd64"}}, options.Platforms))
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			cmd := NewSaveCommand(test.NewFakeCli(&fakeClient{
				imageSaveFunc: tc.imageSaveFunc,
			}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.NilError(t, cmd.Execute())
			if tc.deferredFunc != nil {
				tc.deferredFunc()
			}
		})
	}
}
