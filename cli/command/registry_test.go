package command_test

import (
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/secretsengine"
	"github.com/moby/moby/api/pkg/authconfig"
	"github.com/moby/moby/api/types/registry"
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

	tmpDir := t.TempDir()
	cfg := configfile.New(filepath.Join(tmpDir, "cli-config.json"))
	for _, authConfig := range testAuthConfigs {
		assert.Check(t, cfg.GetCredentialsStore(authConfig.ServerAddress).Store(configtypes.AuthConfig{
			Username:      authConfig.Username,
			Password:      authConfig.Password,
			ServerAddress: authConfig.ServerAddress,

			// TODO(thaJeztah): Are these expected to be included?
			Auth:          authConfig.Auth,
			IdentityToken: authConfig.IdentityToken,
			RegistryToken: authConfig.RegistryToken,
		}))
	}
	for _, tc := range testCases {
		serverAddress := tc.inputServerAddress
		authCfg, err := command.GetDefaultAuthConfig(cfg, tc.checkCredStore, serverAddress, serverAddress == "https://index.docker.io/v1/")
		assert.NilError(t, err)
		assert.Check(t, is.DeepEqual(tc.expectedAuthConfig, authCfg))
	}
}

type fakeAuthProvider struct {
	hosts []string
}

func (p *fakeAuthProvider) GetAuthConfig(registryHostname string) (configtypes.AuthConfig, error) {
	p.hosts = append(p.hosts, registryHostname)
	return configtypes.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: registryHostname}, nil
}

type authProviderCli struct {
	command.Cli
	auth *fakeAuthProvider
}

func (c *authProviderCli) RegistryAuthProvider() secretsengine.AuthProvider {
	return c.auth
}

func TestEncodedRegistryAuthUsesCLIAuthProvider(t *testing.T) {
	dockerCLI := &authProviderCli{auth: &fakeAuthProvider{}}

	actual, err := command.EncodedRegistryAuth(dockerCLI, "dhi.io/python")
	assert.NilError(t, err)
	expected, err := authconfig.Encode(registry.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: "dhi.io"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
	assert.Check(t, is.DeepEqual(dockerCLI.auth.hosts, []string{"dhi.io"}))
}

func TestGetDefaultAuthConfig_HelperError(t *testing.T) {
	cfg := configfile.New("filename")
	cfg.CredentialsStore = "fake-does-not-exist"

	const serverAddress = "test-server-address"
	expectedAuthConfig := registry.AuthConfig{
		ServerAddress: serverAddress,
	}
	const isDefaultRegistry = false // registry is not "https://index.docker.io/v1/"
	authCfg, err := command.GetDefaultAuthConfig(cfg, true, serverAddress, isDefaultRegistry)
	assert.Check(t, is.DeepEqual(expectedAuthConfig, authCfg))
	assert.Check(t, is.ErrorContains(err, "docker-credential-fake-does-not-exist"))
}
