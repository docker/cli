package volume

import (
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func TestUpdateCmd(t *testing.T) {
	cmd := newUpdateCommand(
		test.NewFakeCli(&fakeClient{}),
	)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()

	assert.ErrorContains(t, err, "requires exactly 1 argument")
}
