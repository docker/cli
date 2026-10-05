package registry

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/registry"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRunLogoutDockerIORemovesHubCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := configfile.New(filepath.Join(tmpDir, "config.json"))
	cfg.AuthConfigs = map[string]configtypes.AuthConfig{
		registry.IndexServer: {
			Username:      "user",
			Password:      "pass",
			ServerAddress: registry.IndexServer,
		},
		"example.com": {
			Username:      "other",
			Password:      "secret",
			ServerAddress: "example.com",
		},
	}
	cli := test.NewFakeCli(nil)
	cli.SetConfigFile(cfg)

	assert.NilError(t, runLogout(context.Background(), cli, "docker.io"))

	_, hubOK := cfg.AuthConfigs[registry.IndexServer]
	assert.Check(t, !hubOK)
	_, otherOK := cfg.AuthConfigs["example.com"]
	assert.Check(t, otherOK)
	assert.Check(t, is.Contains(cli.OutBuffer().String(), "Removing login credentials for "+registry.IndexServer))
}
