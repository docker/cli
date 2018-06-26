package stack

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/assert"
)

func TestDeployWithEmptyName(t *testing.T) {
	cmd := newDeployCommand(test.NewFakeCli(&fakeClient{}), nil)
	cmd.SetArgs([]string{"'   '"})
	cmd.SetOutput(ioutil.Discard)

	assert.ErrorContains(t, cmd.Execute(), `invalid stack name: "'   '"`)
}
