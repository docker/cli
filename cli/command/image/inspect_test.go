package image

import (
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/image"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
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
			expectedError: "requires at least 1 argument",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd := newInspectCommand(test.NewFakeCli(&fakeClient{}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewInspectCommandSuccess(t *testing.T) {
	imageInspectInvocationCount := 0
	testCases := []struct {
		name             string
		args             []string
		imageCount       int
		imageInspectFunc func(img string) (image.InspectResponse, []byte, error)
	}{
		{
			name:       "simple",
			args:       []string{"image"},
			imageCount: 1,
			imageInspectFunc: func(img string) (image.InspectResponse, []byte, error) {
				imageInspectInvocationCount++
				assert.Check(t, is.Equal("image", img))
				return image.InspectResponse{}, nil, nil
			},
		},
		{
			name:       "format",
			imageCount: 1,
			args:       []string{"--format='{{.ID}}'", "image"},
			imageInspectFunc: func(img string) (image.InspectResponse, []byte, error) {
				imageInspectInvocationCount++
				return image.InspectResponse{ID: img}, nil, nil
			},
		},
		{
			name:       "simple-many",
			args:       []string{"image1", "image2"},
			imageCount: 2,
			imageInspectFunc: func(img string) (image.InspectResponse, []byte, error) {
				imageInspectInvocationCount++
				if imageInspectInvocationCount == 1 {
					assert.Check(t, is.Equal("image1", img))
				} else {
					assert.Check(t, is.Equal("image2", img))
				}
				return image.InspectResponse{}, nil, nil
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			imageInspectInvocationCount = 0
			cli := test.NewFakeCli(&fakeClient{imageInspectFunc: tc.imageInspectFunc})
			cmd := newInspectCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("inspect-command-success.%s.golden", tc.name))
			assert.Check(t, is.Equal(imageInspectInvocationCount, tc.imageCount))
		})
	}
}
