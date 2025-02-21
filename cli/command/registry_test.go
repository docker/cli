package command_test // import "docker.com/cli/v28/cli/command"

import (
	"testing"

	"github.com/docker/cli/v28/cli/command"
	"github.com/docker/cli/v28/cli/config/configfile"
	configtypes "github.com/docker/cli/v28/cli/config/types"
	"github.com/docker/docker/api/types/registry"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

var testAuthConfigs = []registry.AuthConfig{
	{
		ServerAddress: "https://index.docker.io/v1/",
		Username:      "u0",
		Password:      "p0",
	},
	{
		ServerAddress: "server1.io",
		Username:      "u1",
		Password:      "p1",
	},
}

func TestGetDefaultAuthConfig(t *testing.T) {
	testCases := []struct {
		checkCredStore     bool
		inputServerAddress string
		expectedAuthConfig registry.AuthConfig
	}{
		{
			checkCredStore:     false,
			inputServerAddress: "",
			expectedAuthConfig: registry.AuthConfig{
				ServerAddress: "",
				Username:      "",
				Password:      "",
			},
		},
		{
			checkCredStore:     true,
			inputServerAddress: testAuthConfigs[0].ServerAddress,
			expectedAuthConfig: testAuthConfigs[0],
		},
		{
			checkCredStore:     true,
			inputServerAddress: testAuthConfigs[1].ServerAddress,
			expectedAuthConfig: testAuthConfigs[1],
		},
		{
			checkCredStore:     true,
			inputServerAddress: "https://" + testAuthConfigs[1].ServerAddress,
			expectedAuthConfig: testAuthConfigs[1],
		},
	}
	cfg := configfile.New("filename")
	for _, authconfig := range testAuthConfigs {
		assert.Check(t, cfg.GetCredentialsStore(authconfig.ServerAddress).Store(configtypes.AuthConfig(authconfig)))
	}
	for _, tc := range testCases {
		serverAddress := tc.inputServerAddress
		authconfig, err := command.GetDefaultAuthConfig(cfg, tc.checkCredStore, serverAddress, serverAddress == "https://index.docker.io/v1/")
		assert.NilError(t, err)
		assert.Check(t, is.DeepEqual(tc.expectedAuthConfig, authconfig))
	}
}

func TestGetDefaultAuthConfig_HelperError(t *testing.T) {
	cfg := configfile.New("filename")
	cfg.CredentialsStore = "fake-does-not-exist"

	const serverAddress = "test-server-address"
	expectedAuthConfig := registry.AuthConfig{
		ServerAddress: serverAddress,
	}
	const isDefaultRegistry = false // registry is not "https://index.docker.io/v1/"
	authconfig, err := command.GetDefaultAuthConfig(cfg, true, serverAddress, isDefaultRegistry)
	assert.Check(t, is.DeepEqual(expectedAuthConfig, authconfig))
	assert.Check(t, is.ErrorContains(err, "docker-credential-fake-does-not-exist"))
}
