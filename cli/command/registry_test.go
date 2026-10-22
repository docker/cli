package command_test

import (
	"bytes"
	"path"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
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

func noDesktopSession(string) (configtypes.AuthConfig, bool) {
	return configtypes.AuthConfig{}, false
}

func TestRetrieveAuthTokenFromImageDesktopSession(t *testing.T) {
	const configFileContent = `{"auths": {
		"https://index.docker.io/v1/": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"ghcr.io": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="}
	}
}`
	cfg := configfile.ConfigFile{}
	assert.NilError(t, cfg.LoadFromReader(bytes.NewReader([]byte(configFileContent))))

	var lookups []string
	command.SetDesktopAuthConfig(t, func(registryHostname string) (configtypes.AuthConfig, bool) {
		lookups = append(lookups, registryHostname)
		if registryHostname != "docker.io" {
			return configtypes.AuthConfig{}, false
		}
		return configtypes.AuthConfig{
			Username:      "hubuser",
			Password:      "desktop-token",
			ServerAddress: "https://index.docker.io/v1/",
		}, true
	})

	tests := []struct {
		imageRef string
		expected registry.AuthConfig
	}{
		{
			imageRef: "ubuntu",
			expected: registry.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: "https://index.docker.io/v1/"},
		},
		{
			imageRef: "ghcr.io/foo/bar",
			expected: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "ghcr.io"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.imageRef, func(t *testing.T) {
			actual, err := command.RetrieveAuthTokenFromImage(&cfg, tc.imageRef)
			assert.NilError(t, err)
			expected, err := authconfig.Encode(tc.expected)
			assert.NilError(t, err)
			assert.Equal(t, actual, expected)
		})
	}
	assert.Check(t, is.DeepEqual(lookups, []string{"docker.io", "ghcr.io"}))
}

func TestRetrieveAuthTokenFromImageEnvTakesPrecedenceOverDesktop(t *testing.T) {
	t.Setenv(configfile.DockerEnvConfigKey, `{"auths":{"https://index.docker.io/v1/":{"auth":"ZW52dXNlcjplbnZwYXNz"}}}`)

	var lookups []string
	command.SetDesktopAuthConfig(t, func(registryHostname string) (configtypes.AuthConfig, bool) {
		lookups = append(lookups, registryHostname)
		return configtypes.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: registryHostname}, true
	})
	cfg := configfile.ConfigFile{}

	actual, err := command.RetrieveAuthTokenFromImage(&cfg, "ubuntu")
	assert.NilError(t, err)
	expected, err := authconfig.Encode(registry.AuthConfig{Username: "envuser", Password: "envpass", ServerAddress: "https://index.docker.io/v1/"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
	assert.Check(t, is.Len(lookups, 0), "Docker Desktop must not be asked for a registry in DOCKER_AUTH_CONFIG")

	actual, err = command.RetrieveAuthTokenFromImage(&cfg, "dhi.io/python")
	assert.NilError(t, err)
	expected, err = authconfig.Encode(registry.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: "dhi.io"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
	assert.Check(t, is.DeepEqual(lookups, []string{"dhi.io"}))
}

func TestRetrieveAuthTokenFromImage(t *testing.T) {
	command.SetDesktopAuthConfig(t, noDesktopSession)
	// configFileContent contains a plain-text "username:password", as stored by
	// the plain-text store;
	// https://github.com/docker/cli/blob/v28.0.4/cli/config/configfile/file.go#L218-L229
	const configFileContent = `{"auths": {
		"https://index.docker.io/v1/": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"[::1]": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"[::1]:5000": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"127.0.0.1": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"127.0.0.1:5000": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"localhost": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"localhost:5000": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"registry-1.docker.io": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"registry.hub.docker.com": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="}
	}
}`
	cfg := configfile.ConfigFile{}
	err := cfg.LoadFromReader(bytes.NewReader([]byte(configFileContent)))
	assert.NilError(t, err)

	remoteRefs := []string{
		"ubuntu",
		"ubuntu:latest",
		"ubuntu:latest@sha256:72297848456d5d37d1262630108ab308d3e9ec7ed1c3286a32fe09856619a782",
		"ubuntu@sha256:72297848456d5d37d1262630108ab308d3e9ec7ed1c3286a32fe09856619a782",
		"library/ubuntu",
		"library/ubuntu:latest",
		"library/ubuntu:latest@sha256:72297848456d5d37d1262630108ab308d3e9ec7ed1c3286a32fe09856619a782",
		"library/ubuntu@sha256:72297848456d5d37d1262630108ab308d3e9ec7ed1c3286a32fe09856619a782",
	}

	tests := []struct {
		prefix          string
		expectedAddress string
		expectedAuthCfg registry.AuthConfig
	}{
		{
			prefix:          "",
			expectedAddress: "https://index.docker.io/v1/",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "https://index.docker.io/v1/"},
		},
		{
			prefix:          "docker.io",
			expectedAddress: "https://index.docker.io/v1/",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "https://index.docker.io/v1/"},
		},
		{
			prefix:          "index.docker.io",
			expectedAddress: "https://index.docker.io/v1/",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "https://index.docker.io/v1/"},
		},
		{
			// FIXME(thaJeztah): registry-1.docker.io (the actual registry) is the odd one out, and is stored separate from other URLs used for docker hub's registry
			prefix:          "registry-1.docker.io",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "registry-1.docker.io"},
		},
		{
			// FIXME(thaJeztah): registry.hub.docker.com is stored separate from other URLs used for docker hub's registry
			prefix:          "registry.hub.docker.com",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "registry.hub.docker.com"},
		},
		{
			prefix:          "[::1]",
			expectedAddress: "[::1]",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "[::1]"},
		},
		{
			prefix:          "[::1]:5000",
			expectedAddress: "[::1]:5000",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "[::1]:5000"},
		},
		{
			prefix:          "127.0.0.1",
			expectedAddress: "127.0.0.1",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "127.0.0.1"},
		},
		{
			prefix:          "localhost",
			expectedAddress: "localhost",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "localhost"},
		},
		{
			prefix:          "localhost:5000",
			expectedAddress: "localhost:5000",
			expectedAuthCfg: registry.AuthConfig{Username: "username", Password: "password", ServerAddress: "localhost:5000"},
		},
		{
			prefix:          "no-auth.example.com",
			expectedAuthCfg: registry.AuthConfig{},
		},
	}

	for _, tc := range tests {
		tcName := tc.prefix
		if tc.prefix == "" {
			tcName = "no-prefix"
		}
		t.Run(tcName, func(t *testing.T) {
			for _, remoteRef := range remoteRefs {
				imageRef := path.Join(tc.prefix, remoteRef)
				actual, err := command.RetrieveAuthTokenFromImage(&cfg, imageRef)
				assert.NilError(t, err)
				expectedAuthCfg, err := authconfig.Encode(tc.expectedAuthCfg)
				assert.NilError(t, err)
				assert.Equal(t, actual, expectedAuthCfg)
			}
		})
	}
}
