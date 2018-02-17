package image

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
		{
			name:          "trust-error",
			args:          []string{"--disable-content-trust=false", "image:repo"},
			expectedError: "you are not authorized to perform this operation: server returned 401.",
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{imagePushFunc: tc.imagePushFunc})
		cmd := NewPushCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewPushCommandSuccess(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{
			name: "simple",
			args: []string{"image:tag"},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			imagePushFunc: func(ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		})
		cmd := NewPushCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		assert.NoError(t, cmd.Execute())
	}
}
