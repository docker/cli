package registryauth

import (
	"bytes"
	"sync"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/secretsengine"
	"github.com/moby/moby/api/pkg/authconfig"
	registrytypes "github.com/moby/moby/api/types/registry"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type recordingAuthProvider struct {
	mu    sync.Mutex
	hosts []string
}

func (p *recordingAuthProvider) GetAuthConfig(registryHostname string) (configtypes.AuthConfig, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hosts = append(p.hosts, registryHostname)
	return configtypes.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: registryHostname}, nil
}

type configOnlyCli struct {
	cfg *configfile.ConfigFile
}

func (c *configOnlyCli) ConfigFile() *configfile.ConfigFile {
	return c.cfg
}

type fakeCli struct {
	configOnlyCli
	auth *recordingAuthProvider
}

func (c *fakeCli) RegistryAuthProvider() secretsengine.AuthProvider {
	return c.auth
}

func noEnvAuth(string) (configtypes.AuthConfig, bool) {
	return configtypes.AuthConfig{}, false
}

func loadConfig(t *testing.T) *configfile.ConfigFile {
	t.Helper()
	const configFileContent = `{"auths": {
		"https://index.docker.io/v1/": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"localhost:5000": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="}
	}
}`
	cfg := &configfile.ConfigFile{}
	assert.NilError(t, cfg.LoadFromReader(bytes.NewReader([]byte(configFileContent))))
	return cfg
}

func TestEncodedAuthUsesCLIAuthProvider(t *testing.T) {
	t.Parallel()
	dockerCLI := &fakeCli{configOnlyCli: configOnlyCli{cfg: configfile.New("")}, auth: &recordingAuthProvider{}}

	actual, err := EncodedAuth(dockerCLI, "dhi.io/python")
	assert.NilError(t, err)
	expected, err := authconfig.Encode(registrytypes.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: "dhi.io"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
	assert.Check(t, is.DeepEqual(dockerCLI.auth.hosts, []string{"dhi.io"}))
}

func TestEncodedAuthWithoutAuthProviderUsesConfigFile(t *testing.T) {
	t.Parallel()
	actual, err := EncodedAuth(&configOnlyCli{cfg: loadConfig(t)}, "ubuntu")
	assert.NilError(t, err)
	expected, err := authconfig.Encode(registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "https://index.docker.io/v1/"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
}

func TestEncodedAuthEnvTakesPrecedence(t *testing.T) {
	t.Parallel()
	envAuth := func(registryHostname string) (configtypes.AuthConfig, bool) {
		return configtypes.AuthConfig{Username: "envuser", Password: "envpass", ServerAddress: registryHostname}, true
	}
	auth := &recordingAuthProvider{}

	actual, err := encodedAuth(envAuth, auth, "ubuntu")
	assert.NilError(t, err)
	expected, err := authconfig.Encode(registrytypes.AuthConfig{Username: "envuser", Password: "envpass", ServerAddress: "docker.io"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
	assert.Check(t, is.Len(auth.hosts, 0), "the auth provider must not be asked for a registry in DOCKER_AUTH_CONFIG")
}

func TestEnvAuthConfig(t *testing.T) {
	t.Setenv(configfile.DockerEnvConfigKey, `{"auths":{"https://index.docker.io/v1/":{"auth":"ZW52dXNlcjplbnZwYXNz"}}}`)

	got, ok := envAuthConfig("docker.io")
	assert.Check(t, ok)
	assert.Check(t, is.DeepEqual(got, configtypes.AuthConfig{Username: "envuser", Password: "envpass", ServerAddress: "https://index.docker.io/v1/"}))

	_, ok = envAuthConfig("ghcr.io")
	assert.Check(t, !ok)
}

func TestEncodedAuthFromConfigFile(t *testing.T) {
	t.Parallel()
	cfg := loadConfig(t)

	tests := []struct {
		image    string
		expected registrytypes.AuthConfig
	}{
		{image: "ubuntu", expected: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "https://index.docker.io/v1/"}},
		{image: "localhost:5000/ubuntu", expected: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "localhost:5000"}},
		{image: "no-auth.example.com/ubuntu", expected: registrytypes.AuthConfig{}},
	}
	for _, tc := range tests {
		t.Run(tc.image, func(t *testing.T) {
			t.Parallel()
			actual, err := encodedAuth(noEnvAuth, cfg, tc.image)
			assert.NilError(t, err)
			expected, err := authconfig.Encode(tc.expected)
			assert.NilError(t, err)
			assert.Check(t, is.Equal(actual, expected))
		})
	}
}
