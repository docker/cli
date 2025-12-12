package image

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/image"
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

func TestImagesFilterDangling(t *testing.T) {
	// Create test images with different states
	items := []image.Summary{
		{
			ID:          "sha256:87428fc522803d31065e7bce3cf03fe475096631e5e07bbd7a0fde60c4cf25c7",
			RepoTags:    []string{"myimage:latest"},
			RepoDigests: []string{"myimage@sha256:abc123"},
		},
		{
			ID:          "sha256:0263829989b6fd954f72baaf2fc64bc2e2f01d692d4de72986ea808f6e99813f",
			RepoTags:    []string{},
			RepoDigests: []string{},
		},
		{
			ID:          "sha256:a3a5e715f0cc574a73c3f9bebb6bc24f32ffd5b67b387244c2c909da779a1478",
			RepoTags:    []string{},
			RepoDigests: []string{"image@sha256:a3a5e715f0cc574a73c3f9bebb6bc24f32ffd5b67b387244c2c909da779a1478"},
		},
	}

	testCases := []struct {
		name          string
		args          []string
		imageListFunc func(options client.ImageListOptions) (client.ImageListResult, error)
	}{
		{
			name: "dangling-true",
			args: []string{"-f", "dangling=true"},
			imageListFunc: func(options client.ImageListOptions) (client.ImageListResult, error) {
				// Verify the filter is passed to the API
				assert.Check(t, options.Filters["dangling"]["true"])
				// dangling=true is handled on the server side and returns only dangling images
				return client.ImageListResult{Items: []image.Summary{items[1], items[2]}}, nil
			},
		},
		{
			name: "dangling-false",
			args: []string{"-f", "dangling=false"},
			imageListFunc: func(options client.ImageListOptions) (client.ImageListResult, error) {
				// Verify the filter is passed to the API
				assert.Check(t, options.Filters["dangling"]["false"])
				// Return all images including dangling
				return client.ImageListResult{Items: slices.Clone(items)}, nil
			},
		},
		{
			name: "no-dangling-filter",
			args: []string{},
			imageListFunc: func(options client.ImageListOptions) (client.ImageListResult, error) {
				// Verify no dangling filter is passed to the API
				_, exists := options.Filters["dangling"]
				assert.Check(t, !exists)
				// Return all images including dangling
				return client.ImageListResult{Items: slices.Clone(items)}, nil
			},
		},
		{
			name: "all-flag",
			args: []string{"--all"},
			imageListFunc: func(options client.ImageListOptions) (client.ImageListResult, error) {
				// Verify the All flag is set
				assert.Check(t, options.All)
				// Return all images including dangling
				return client.ImageListResult{Items: slices.Clone(items)}, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageListFunc: tc.imageListFunc})
			cmd := newImagesCommand(cli)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("list-command-filter-dangling.%s.golden", tc.name))
		})
	}
}

func nilToEmptySlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
