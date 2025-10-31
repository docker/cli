package image

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNewImagesCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
		imageListFunc func(options client.ImageListOptions) (client.ImageListResult, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{"arg1", "arg2"},
			expectedError: "requires at most 1 argument",
		},
		{
			name:          "failed-list",
			expectedError: "something went wrong",
			imageListFunc: func(options client.ImageListOptions) (client.ImageListResult, error) {
				return client.ImageListResult{}, errors.New("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newImagesCommand(test.NewFakeCli(&fakeClient{imageListFunc: tc.imageListFunc}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(nilToEmptySlice(tc.args))
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewImagesCommandSuccess(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		imageFormat   string
		imageListFunc func(options client.ImageListOptions) (client.ImageListResult, error)
	}{
		{
			name: "simple",
		},
		{
			name:        "format",
			imageFormat: "raw",
		},
		{
			name:        "quiet-format",
			args:        []string{"-q"},
			imageFormat: "table",
		},
		{
			name: "match-name",
			args: []string{"image"},
			imageListFunc: func(options client.ImageListOptions) (client.ImageListResult, error) {
				assert.Check(t, options.Filters["reference"]["image"])
				return client.ImageListResult{}, nil
			},
		},
		{
			name: "filters",
			args: []string{"--filter", "name=value"},
			imageListFunc: func(options client.ImageListOptions) (client.ImageListResult, error) {
				assert.Check(t, options.Filters["name"]["value"])
				return client.ImageListResult{}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageListFunc: tc.imageListFunc})
			cli.SetConfigFile(&configfile.ConfigFile{ImagesFormat: tc.imageFormat})
			cmd := newImagesCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(nilToEmptySlice(tc.args))
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("list-command-success.%s.golden", tc.name))
		})
	}
}

func TestNewListCommandAlias(t *testing.T) {
	cmd := newListCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetArgs([]string{""})
	assert.Check(t, cmd.HasAlias("list"))
	assert.Check(t, !cmd.HasAlias("other"))
}

func TestNewListCommandAmbiguous(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	cmd := newImagesCommand(cli)
	cmd.SetOut(io.Discard)

	// Set the Use field to mimic that the command was called as "docker images",
	// not "docker image ls".
	cmd.Use = "images"
	cmd.SetArgs([]string{"ls"})
	err := cmd.Execute()
	assert.NilError(t, err)
	golden.Assert(t, cli.ErrBuffer().String(), "list-command-ambiguous.golden")
}

func nilToEmptySlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
