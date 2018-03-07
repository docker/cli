package image

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
)

type notFound struct {
	imageID string
}

func (n notFound) Error() string {
	return fmt.Sprintf("Error: No such image: %s", n.imageID)
}

func (n notFound) NotFound() bool {
	return true
}

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
		imageRemoveFunc func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	}{
		{
			name:          "wrong args",
			expectedError: "requires at least 1 argument.",
		},
		{
			name:          "ImageRemove fail with force option",
			args:          []string{"-f", "image1"},
			expectedError: "error removing image",
			imageRemoveFunc: func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
				assert.Check(t, is.Equal("image1", image))
				return []types.ImageDeleteResponseItem{}, errors.Errorf("error removing image")
			},
		},
		{
			name:          "ImageRemove fail",
			args:          []string{"arg1"},
			expectedError: "error removing image",
			imageRemoveFunc: func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
				assert.Check(t, !options.Force)
				assert.Check(t, options.PruneChildren)
				return []types.ImageDeleteResponseItem{}, errors.Errorf("error removing image")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewRemoveCommand(test.NewFakeCli(&fakeClient{
				imageRemoveFunc: tc.imageRemoveFunc,
			}))
			cmd.SetOutput(ioutil.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewRemoveCommandSuccess(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		imageRemoveFunc func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
		expectedStderr  string
	}{
		{
			name: "Image Deleted",
			args: []string{"image1"},
			imageRemoveFunc: func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
				assert.Check(t, is.Equal("image1", image))
				return []types.ImageDeleteResponseItem{{Deleted: image}}, nil
			},
		},
		{
			name: "Image not found with force option",
			args: []string{"-f", "image1"},
			imageRemoveFunc: func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
				assert.Check(t, is.Equal("image1", image))
				assert.Check(t, is.Equal(true, options.Force))
				return []types.ImageDeleteResponseItem{}, notFound{"image1"}
			},
			expectedStderr: "Error: No such image: image1\n",
		},

		{
			name: "Image Untagged",
			args: []string{"image1"},
			imageRemoveFunc: func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
				assert.Check(t, is.Equal("image1", image))
				return []types.ImageDeleteResponseItem{{Untagged: image}}, nil
			},
		},
		{
			name: "Image Deleted and Untagged",
			args: []string{"image1", "image2"},
			imageRemoveFunc: func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
				if image == "image1" {
					return []types.ImageDeleteResponseItem{{Untagged: image}}, nil
				}
				return []types.ImageDeleteResponseItem{{Deleted: image}}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageRemoveFunc: tc.imageRemoveFunc})
			cmd := NewRemoveCommand(cli)
			cmd.SetOutput(ioutil.Discard)
			cmd.SetArgs(tc.args)
			assert.NilError(t, cmd.Execute())
			assert.Check(t, is.Equal(tc.expectedStderr, cli.ErrBuffer().String()))
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("remove-command-success.%s.golden", tc.name))
		})
	}
}
