package system

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
)

func TestVersionWithoutServer(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serverVersion: func(ctx context.Context) (types.Version, error) {
			return types.Version{}, fmt.Errorf("no server")
		},
	})
	cmd := NewVersionCommand(cli)
	cmd.SetOutput(cli.Err())
	assert.Check(t, is.ErrorContains(cmd.Execute(), ""))
	assert.Check(t, is.Contains(cleanTabs(cli.OutBuffer().String()), "Client:"))
	assert.NotContains(t, cleanTabs(cli.OutBuffer().String()), "Server:")
}

func fakeServerVersion(ctx context.Context) (types.Version, error) {
	return types.Version{
		Version:    "docker-dev",
		APIVersion: api.DefaultVersion,
	}, nil
}

func TestVersionWithOrchestrator(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{serverVersion: fakeServerVersion})
	cli.SetClientInfo(func() command.ClientInfo { return command.ClientInfo{Orchestrator: "swarm"} })
	cmd := NewVersionCommand(cli)
	assert.Check(t, cmd.Execute())
	assert.Check(t, is.Contains(cleanTabs(cli.OutBuffer().String()), "Orchestrator: swarm"))
}

func cleanTabs(line string) string {
	return strings.Join(strings.Fields(line), " ")
}
