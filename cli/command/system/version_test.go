package system

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
)

func TestVersionWithoutServer(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serverVersion: func(ctx context.Context) (types.Version, error) {
			return types.Version{}, fmt.Errorf("no server")
		},
	})
	cmd := NewVersionCommand(cli)
	cmd.SetOutput(cli.Err())
	assert.ErrorContains(t, cmd.Execute(), "no server")
	out := cli.OutBuffer().String()
	// TODO: use an assertion like e2e/image/build_test.go:assertBuildOutput()
	// instead of contains/not contains
	assert.Check(t, is.Contains(out, "Client:"))
	assert.Assert(t, !strings.Contains(out, "Server:"), "actual: %s", out)
}

func fakeServerVersion(_ context.Context) (types.Version, error) {
	return types.Version{
		Version:    "docker-dev",
		APIVersion: api.DefaultVersion,
	}, nil
}

func TestVersionWithOrchestrator(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{serverVersion: fakeServerVersion}, test.OrchestratorSwarm)
	cmd := NewVersionCommand(cli)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Contains(cleanTabs(cli.OutBuffer().String()), "Orchestrator: swarm"))
}

func cleanTabs(line string) string {
	return strings.Join(strings.Fields(line), " ")
}
