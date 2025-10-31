package image

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

type notFound struct {
	imageID string
}

func (n notFound) Error() string {
	return "Error: No such image: " + n.imageID
}

func (notFound) NotFound() {}

func TestNewRemoveCommandAlias(t *testing.T) {
	cmd := newImageRemoveCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetArgs([]string{""})
	assert.Check(t, cmd.HasAlias("rmi"))
	assert.Check(t, cmd.HasAlias("remove"))
	assert.Check(t, !cmd.HasAlias("other"))
}

func TestNewRemoveCommandErrors(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		expectedError   string
		imageRemoveFunc func(img string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error)
	}{
		{
			name:          "wrong args",
			expectedError: "requires at least 1 argument",
		},
		{
			name:          "ImageRemove fail with force option",
			args:          []string{"-f", "image1"},
			expectedError: "error removing image",
			imageRemoveFunc: func(img string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
				assert.Check(t, is.Equal("image1", img))
				return client.ImageRemoveResult{}, errors.New("error removing image")
			},
		},
		{
			name:          "ImageRemove fail",
			args:          []string{"arg1"},
			expectedError: "error removing image",
			imageRemoveFunc: func(img string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
				assert.Check(t, !options.Force)
				assert.Check(t, options.PruneChildren)
				return client.ImageRemoveResult{}, errors.New("error removing image")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRemoveCommand(test.NewFakeCli(&fakeClient{
				imageRemoveFunc: tc.imageRemoveFunc,
			}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(nilToEmptySlice(tc.args))
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewRemoveCommandSuccess(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		imageRemoveFunc func(img string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error)
		expectedStderr  string
	}{
		{
			name: "Image Deleted",
			args: []string{"image1"},
			imageRemoveFunc: func(img string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
				assert.Check(t, is.Equal("image1", img))
				return client.ImageRemoveResult{
					Items: []image.DeleteResponse{{Deleted: img}},
				}, nil
			},
		},
		{
			name: "Image not found with force option",
			args: []string{"-f", "image1"},
			imageRemoveFunc: func(img string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
				assert.Check(t, is.Equal("image1", img))
				assert.Check(t, is.Equal(true, options.Force))
				return client.ImageRemoveResult{}, notFound{"image1"}
			},
			expectedStderr: "Error: No such image: image1\n",
		},

		{
			name: "Image Untagged",
			args: []string{"image1"},
			imageRemoveFunc: func(img string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
				assert.Check(t, is.Equal("image1", img))
				return client.ImageRemoveResult{
					Items: []image.DeleteResponse{{Untagged: img}},
				}, nil
			},
		},
		{
			name: "Image Deleted and Untagged",
			args: []string{"image1", "image2"},
			imageRemoveFunc: func(img string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
				if img == "image1" {
					return client.ImageRemoveResult{
						Items: []image.DeleteResponse{{Untagged: img}},
					}, nil
				}
				return client.ImageRemoveResult{
					Items: []image.DeleteResponse{{Deleted: img}},
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageRemoveFunc: tc.imageRemoveFunc})
			cmd := newRemoveCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(nilToEmptySlice(tc.args))
			assert.NilError(t, cmd.Execute())
			assert.Check(t, is.Equal(tc.expectedStderr, cli.ErrBuffer().String()))
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("remove-command-success.%s.golden", tc.name))
		})
	}
}
