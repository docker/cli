package image

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestNewPullCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "wrong-args",
			expectedError: "requires 1 argument",
			args:          []string{},
		},
		{
			name:          "invalid-name",
			expectedError: "invalid reference format: repository name (library/UPPERCASE_REPO) must be lowercase",
			args:          []string{"UPPERCASE_REPO"},
		},
		{
			name:          "all-tags-with-tag",
			expectedError: "tag can't be used with --all-tags/-a",
			args:          []string{"--all-tags", "image:tag"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			cmd := newPullCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
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
		{
			name:        "simple-quiet",
			args:        []string{"--quiet", "image"},
			expectedTag: "image:latest",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				imagePullFunc: func(ref string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
					assert.Check(t, is.Equal(tc.expectedTag, ref), tc.name)
					// FIXME(thaJeztah): how to mock this?
					return fakeStreamResult{ReadCloser: http.NoBody}, nil
				},
			})
			cmd := newPullCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("pull-command-success.%s.golden", tc.name))
		})
	}
}
