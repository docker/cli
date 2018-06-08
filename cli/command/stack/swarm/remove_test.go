package swarm

import (
	"testing"

	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/internal/test"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestRunRemoveWithEmptyName(t *testing.T) {
	client := &fakeClient{}
	dockerCli := test.NewFakeCli(client)

	err := RunRemove(dockerCli, options.Remove{Namespaces: []string{"good", "'   '", "alsogood"}})
	assert.Check(t, is.Error(err, `invalid stack name: "'   '"`))
}
