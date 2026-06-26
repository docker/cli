package image

import (
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCliNewTagCommandErrors(t *testing.T) {
	testCases := [][]string{
		{},
		{"image1"},
		{"image1", "image2", "image3"},
	}
	expectedError := "'tag' requires 2 arguments"
	for _, args := range testCases {
		cmd := newTagCommand(test.NewFakeCli(&fakeClient{}))
		cmd.SetArgs(args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), expectedError)
	}
}

func TestCliNewTagCommand(t *testing.T) {
	cmd := newTagCommand(
		test.NewFakeCli(&fakeClient{
			imageTagFunc: func(options client.ImageTagOptions) (client.ImageTagResult, error) {
				assert.Check(t, is.Equal("image1", options.Source))
				assert.Check(t, is.Equal("image2", options.Target))
				return client.ImageTagResult{}, nil
			},
		}))
	cmd.SetArgs([]string{"image1", "image2"})
	cmd.SetOut(io.Discard)
	assert.NilError(t, cmd.Execute())
	value, _ := cmd.Flags().GetBool("interspersed")
	assert.Check(t, !value)
}
