package swarm

import (
	"testing"

	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestRunServicesWithEmptyName(t *testing.T) {
	client := &fakeClient{}
	dockerCli := test.NewFakeCli(client)

	err := RunServices(dockerCli, options.Services{Namespace: "'   '"})
	assert.Check(t, is.Error(err, `invalid stack name: "'   '"`))
}
