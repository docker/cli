package swarm

import (
	"testing"

	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestRunRemoveWithEmptyName(t *testing.T) {
	client := &fakeClient{}
	dockerCli := test.NewFakeCli(client)

	err := RunRemove(dockerCli, options.Remove{Namespaces: []string{"good", "'   '", "alsogood"}})
	assert.Check(t, is.Error(err, `invalid stack name: "'   '"`))
}
