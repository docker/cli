package image

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func TestNewSaveCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		isTerminal    bool
		expectedError string
		imageSaveFunc func(images []string) (io.ReadCloser, error)
	}{
		{
			name:          "wrong args",
			args:          []string{},
			expectedError: "requires at least 1 argument.",
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
			imageSaveFunc: func(images []string) (io.ReadCloser, error) {
				return ioutil.NopCloser(strings.NewReader("")), errors.Errorf("error saving image")
			},
		},
		{
			name:          "output directory does not exist",
			args:          []string{"-o", "fakedir/out.tar", "arg1"},
			expectedError: "failed to save image: unable to validate output path: directory \"fakedir\" does not exist",
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{imageSaveFunc: tc.imageSaveFunc})
		cli.Out().SetIsTerminal(tc.isTerminal)
		cmd := NewSaveCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewSaveCommandSuccess(t *testing.T) {
	testCases := []struct {
		args          []string
		isTerminal    bool
		imageSaveFunc func(images []string) (io.ReadCloser, error)
		deferredFunc  func()
	}{
		{
			args:       []string{"-o", "save_tmp_file", "arg1"},
			isTerminal: true,
			imageSaveFunc: func(images []string) (io.ReadCloser, error) {
				assert.Assert(t, is.Len(images, 1))
				assert.Check(t, is.Equal("arg1", images[0]))
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
			deferredFunc: func() {
				os.Remove("save_tmp_file")
			},
		},
		{
			args:       []string{"arg1", "arg2"},
			isTerminal: false,
			imageSaveFunc: func(images []string) (io.ReadCloser, error) {
				assert.Assert(t, is.Len(images, 2))
				assert.Check(t, is.Equal("arg1", images[0]))
				assert.Check(t, is.Equal("arg2", images[1]))
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		},
	}
	for _, tc := range testCases {
		cmd := NewSaveCommand(test.NewFakeCli(&fakeClient{
			imageSaveFunc: func(images []string) (io.ReadCloser, error) {
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		}))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		assert.NilError(t, cmd.Execute())
		if tc.deferredFunc != nil {
			tc.deferredFunc()
		}
	}
}
