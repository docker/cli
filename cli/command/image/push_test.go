package image

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
)

func TestNewPushCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
		imagePushFunc func(ref string, options types.ImagePushOptions) (io.ReadCloser, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{},
			expectedError: "requires exactly 1 argument.",
		},
		{
			name:          "invalid-name",
			args:          []string{"UPPERCASE_REPO"},
			expectedError: "invalid reference format: repository name must be lowercase",
		},
		{
			name:          "push-failed",
			args:          []string{"image:repo"},
			expectedError: "Failed to push",
			imagePushFunc: func(ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
				return ioutil.NopCloser(strings.NewReader("")), errors.Errorf("Failed to push")
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{imagePushFunc: tc.imagePushFunc})
		cmd := NewPushCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewPushCommandSuccess(t *testing.T) {
	testCases := []struct {
		name   string
		args   []string
		output string
	}{
		{
			name: "push",
			args: []string{"image:tag"},
		},
		{
			name: "push quiet",
			args: []string{"--quiet", "image:tag"},
			output: `docker.io/library/image:tag
`,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				imagePushFunc: func(ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
					return ioutil.NopCloser(strings.NewReader("")), nil
				},
			})
			cmd := NewPushCommand(cli)
			cmd.SetOutput(cli.OutBuffer())
			cmd.SetArgs(tc.args)
			assert.NilError(t, cmd.Execute())
			if tc.output != "" {
				assert.Equal(t, tc.output, cli.OutBuffer().String())
			}
		})
	}
}
