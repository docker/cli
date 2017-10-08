package image

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
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
		{
			name:          "pull-error",
			args:          []string{"--disable-content-trust=false", "image:tag"},
			expectedError: "you are not authorized to perform this operation: server returned 401.",
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
		name string
		args []string
	}{
		{
			name: "simple",
			args: []string{"image:tag"},
		},
		{
			name: "simple-no-tag",
			args: []string{"image"},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		cmd := NewPullCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		assert.NoError(t, err)
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("pull-command-success.%s.golden", tc.name))
	}
}
