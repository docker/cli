package image

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
	assert.True(t, cmd.HasAlias("rmi"))
	assert.True(t, cmd.HasAlias("remove"))
	assert.False(t, cmd.HasAlias("other"))
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
				assert.Equal(t, "image1", image)
				return []types.ImageDeleteResponseItem{}, errors.Errorf("error removing image")
			},
		},
		{
			name:          "ImageRemove fail",
			args:          []string{"arg1"},
			expectedError: "error removing image",
			imageRemoveFunc: func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
				assert.False(t, options.Force)
				assert.True(t, options.PruneChildren)
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
			testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
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
				assert.Equal(t, "image1", image)
				return []types.ImageDeleteResponseItem{{Deleted: image}}, nil
			},
		},
		{
			name: "Image not found with force option",
			args: []string{"-f", "image1"},
			imageRemoveFunc: func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
				assert.Equal(t, "image1", image)
				assert.Equal(t, true, options.Force)
				return []types.ImageDeleteResponseItem{}, notFound{"image1"}
			},
			expectedStderr: "Error: No such image: image1",
		},

		{
			name: "Image Untagged",
			args: []string{"image1"},
			imageRemoveFunc: func(image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
				assert.Equal(t, "image1", image)
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
			assert.NoError(t, cmd.Execute())
			assert.Equal(t, tc.expectedStderr, cli.ErrBuffer().String())
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("remove-command-success.%s.golden", tc.name))
		})
	}
}
