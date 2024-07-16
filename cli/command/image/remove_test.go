package image

import (
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/image"
	"github.com/pkg/errors"
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

func (n notFound) NotFound() {}

func TestNewRemoveCommandAlias(t *testing.T) {
	cmd := newRemoveCommand(test.NewFakeCli(&fakeClient{}))
	assert.Check(t, cmd.HasAlias("rmi"))
	assert.Check(t, cmd.HasAlias("remove"))
	assert.Check(t, !cmd.HasAlias("other"))
}

func TestNewRemoveCommandErrors(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		expectedError   string
		imageRemoveFunc func(img string, options image.RemoveOptions) ([]image.DeleteResponse, error)
	}{
		{
			name:          "wrong args",
			expectedError: "requires at least 1 argument",
		},
		{
			name:          "ImageRemove fail with force option",
			args:          []string{"-f", "image1"},
			expectedError: "error removing image",
			imageRemoveFunc: func(img string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
				assert.Check(t, is.Equal("image1", img))
				return []image.DeleteResponse{}, errors.Errorf("error removing image")
			},
		},
		{
			name:          "ImageRemove fail",
			args:          []string{"arg1"},
			expectedError: "error removing image",
			imageRemoveFunc: func(img string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
				assert.Check(t, !options.Force)
				assert.Check(t, options.PruneChildren)
				return []image.DeleteResponse{}, errors.Errorf("error removing image")
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewRemoveCommand(test.NewFakeCli(&fakeClient{
				imageRemoveFunc: tc.imageRemoveFunc,
			}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewRemoveCommandSuccess(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		imageRemoveFunc func(img string, options image.RemoveOptions) ([]image.DeleteResponse, error)
		expectedStderr  string
	}{
		{
			name: "Image Deleted",
			args: []string{"image1"},
			imageRemoveFunc: func(img string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
				assert.Check(t, is.Equal("image1", img))
				return []image.DeleteResponse{{Deleted: img}}, nil
			},
		},
		{
			name: "Image not found with force option",
			args: []string{"-f", "image1"},
			imageRemoveFunc: func(img string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
				assert.Check(t, is.Equal("image1", img))
				assert.Check(t, is.Equal(true, options.Force))
				return []image.DeleteResponse{}, notFound{"image1"}
			},
			expectedStderr: "Error: No such image: image1\n",
		},

		{
			name: "Image Untagged",
			args: []string{"image1"},
			imageRemoveFunc: func(img string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
				assert.Check(t, is.Equal("image1", img))
				return []image.DeleteResponse{{Untagged: img}}, nil
			},
		},
		{
			name: "Image Deleted and Untagged",
			args: []string{"image1", "image2"},
			imageRemoveFunc: func(img string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
				if img == "image1" {
					return []image.DeleteResponse{{Untagged: img}}, nil
				}
				return []image.DeleteResponse{{Deleted: img}}, nil
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageRemoveFunc: tc.imageRemoveFunc})
			cmd := NewRemoveCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.NilError(t, cmd.Execute())
			assert.Check(t, is.Equal(tc.expectedStderr, cli.ErrBuffer().String()))
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("remove-command-success.%s.golden", tc.name))
		})
	}
}
