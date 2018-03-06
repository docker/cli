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
)

func TestNewInspectCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "wrong-args",
			args:          []string{},
			expectedError: "requires at least 1 argument.",
		},
	}
	for _, tc := range testCases {
		cmd := newInspectCommand(test.NewFakeCli(&fakeClient{}))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewInspectCommandSuccess(t *testing.T) {
	imageInspectInvocationCount := 0
	testCases := []struct {
		name             string
		args             []string
		imageCount       int
		imageInspectFunc func(image string) (types.ImageInspect, []byte, error)
	}{
		{
			name:       "simple",
			args:       []string{"image"},
			imageCount: 1,
			imageInspectFunc: func(image string) (types.ImageInspect, []byte, error) {
				imageInspectInvocationCount++
				assert.Check(t, is.Equal("image", image))
				return types.ImageInspect{}, nil, nil
			},
		},
		{
			name:       "format",
			imageCount: 1,
			args:       []string{"--format='{{.ID}}'", "image"},
			imageInspectFunc: func(image string) (types.ImageInspect, []byte, error) {
				imageInspectInvocationCount++
				return types.ImageInspect{ID: image}, nil, nil
			},
		},
		{
			name:       "simple-many",
			args:       []string{"image1", "image2"},
			imageCount: 2,
			imageInspectFunc: func(image string) (types.ImageInspect, []byte, error) {
				imageInspectInvocationCount++
				if imageInspectInvocationCount == 1 {
					assert.Check(t, is.Equal("image1", image))
				} else {
					assert.Check(t, is.Equal("image2", image))
				}
				return types.ImageInspect{}, nil, nil
			},
		},
	}
	for _, tc := range testCases {
		imageInspectInvocationCount = 0
		cli := test.NewFakeCli(&fakeClient{imageInspectFunc: tc.imageInspectFunc})
		cmd := newInspectCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		assert.NilError(t, err)
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("inspect-command-success.%s.golden", tc.name))
		assert.Check(t, is.Equal(imageInspectInvocationCount, tc.imageCount))
	}
}
