package system

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
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
	assert.Error(t, cmd.Execute())
	assert.Contains(t, cleanTabs(cli.OutBuffer().String()), "Client:")
	assert.NotContains(t, cleanTabs(cli.OutBuffer().String()), "Server:")
}

func fakeServerVersion(ctx context.Context) (types.Version, error) {
	return types.Version{
		Version:    "docker-dev",
		APIVersion: api.DefaultVersion,
	}, nil
}

func TestVersionWithDefaultOrchestrator(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{serverVersion: fakeServerVersion})
	cmd := NewVersionCommand(cli)
	assert.NoError(t, cmd.Execute())
	assert.Contains(t, cleanTabs(cli.OutBuffer().String()), "Orchestrator: swarm")
}

func TestVersionWithOverridenOrchestrator(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{serverVersion: fakeServerVersion})
	config := configfile.New("configfile")
	config.Orchestrator = "Kubernetes"
	cli.SetConfigFile(config)
	cmd := NewVersionCommand(cli)
	assert.NoError(t, cmd.Execute())
	assert.Contains(t, cleanTabs(cli.OutBuffer().String()), "Orchestrator: kubernetes")
}

func cleanTabs(line string) string {
	return strings.Join(strings.Fields(line), " ")
}
