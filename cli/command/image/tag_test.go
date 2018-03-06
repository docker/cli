package image

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestCliNewTagCommandErrors(t *testing.T) {
	testCases := [][]string{
		{},
		{"image1"},
		{"image1", "image2", "image3"},
	}
	expectedError := "\"tag\" requires exactly 2 arguments."
	for _, args := range testCases {
		cmd := NewTagCommand(test.NewFakeCli(&fakeClient{}))
		cmd.SetArgs(args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), expectedError)
	}
}

func TestCliNewTagCommand(t *testing.T) {
	cmd := NewTagCommand(
		test.NewFakeCli(&fakeClient{
			imageTagFunc: func(image string, ref string) error {
				assert.Check(t, is.Equal("image1", image))
				assert.Check(t, is.Equal("image2", ref))
				return nil
			},
		}))
	cmd.SetArgs([]string{"image1", "image2"})
	cmd.SetOutput(ioutil.Discard)
	assert.NilError(t, cmd.Execute())
	value, _ := cmd.Flags().GetBool("interspersed")
	assert.Check(t, !value)
}
