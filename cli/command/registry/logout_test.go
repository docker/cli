package registry

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func TestRunLogoutMixedCaseServerAddress(t *testing.T) {
	cfg := configfile.New(filepath.Join(t.TempDir(), "config.json"))
	cli := test.NewFakeCli(nil)
	cli.SetConfigFile(cfg)

	const serverAddress = "myregistry.example.com"
	assert.NilError(t, cfg.GetCredentialsStore(serverAddress).Store(configtypes.AuthConfig{
		Username:      "my-username",
		Password:      "my-password",
		ServerAddress: serverAddress,
	}))

	assert.NilError(t, runLogout(context.Background(), cli, "MyRegistry.Example.com"))
	credentials, err := cfg.GetAllCredentials()
	assert.NilError(t, err)
	assert.DeepEqual(t, credentials, map[string]configtypes.AuthConfig{})
}

func TestRunLogoutUpperCaseServerAddress(t *testing.T) {
	cfg := configfile.New(filepath.Join(t.TempDir(), "config.json"))
	cli := test.NewFakeCli(nil)
	cli.SetConfigFile(cfg)

	const serverAddress = "MYREGISTRY.EXAMPLE.COM"
	assert.NilError(t, cfg.GetCredentialsStore(serverAddress).Store(configtypes.AuthConfig{
		Username:      "my-username",
		Password:      "my-password",
		ServerAddress: serverAddress,
	}))

	assert.NilError(t, runLogout(context.Background(), cli, serverAddress))
	credentials, err := cfg.GetAllCredentials()
	assert.NilError(t, err)
	assert.DeepEqual(t, credentials, map[string]configtypes.AuthConfig{})
}
