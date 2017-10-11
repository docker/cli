package image

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/stretchr/testify/assert"
)

func TestNewPullCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "wrong-args",
			expectedError: "requires exactly 1 argument.",
			args:          []string{},
		},
		{
			name:          "invalid-name",
			expectedError: "invalid reference format: repository name must be lowercase",
			args:          []string{"UPPERCASE_REPO"},
		},
		{
			name:          "all-tags-with-tag",
			expectedError: "tag can't be used with --all-tags/-a",
			args:          []string{"--all-tags", "image:tag"},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		cmd := NewPullCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewPullCommandSuccess(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		expectedTag string
	}{
		{
			name:        "simple",
			args:        []string{"image:tag"},
			expectedTag: "image:tag",
		},
		{
			name:        "simple-no-tag",
			args:        []string{"image"},
			expectedTag: "image:latest",
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			imagePullFunc: func(ref string, options types.ImagePullOptions) (io.ReadCloser, error) {
				assert.Equal(t, tc.expectedTag, ref, tc.name)
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		})
		cmd := NewPullCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		assert.NoError(t, err)
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("pull-command-success.%s.golden", tc.name))
	}
}
