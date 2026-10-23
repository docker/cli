package registryauth

import (
	"bytes"
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
	hosts []string
}

func (p *recordingAuthProvider) GetAuthConfig(registryHostname string) (configtypes.AuthConfig, error) {
	p.hosts = append(p.hosts, registryHostname)
	return configtypes.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: registryHostname}, nil
}

type fakeCli struct {
	cfg  *configfile.ConfigFile
	auth *recordingAuthProvider
}

func (c *fakeCli) ConfigFile() *configfile.ConfigFile {
	return c.cfg
}

func (c *fakeCli) RegistryAuthProvider() secretsengine.AuthProvider {
	return c.auth
}

func TestEncodedAuthUsesCLIAuthProvider(t *testing.T) {
	dockerCLI := &fakeCli{cfg: configfile.New(""), auth: &recordingAuthProvider{}}

	actual, err := EncodedAuth(dockerCLI, "dhi.io/python")
	assert.NilError(t, err)
	expected, err := authconfig.Encode(registrytypes.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: "dhi.io"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
	assert.Check(t, is.DeepEqual(dockerCLI.auth.hosts, []string{"dhi.io"}))
}

func TestEncodedAuthEnvTakesPrecedence(t *testing.T) {
	t.Setenv(configfile.DockerEnvConfigKey, `{"auths":{"https://index.docker.io/v1/":{"auth":"ZW52dXNlcjplbnZwYXNz"}}}`)
	auth := &recordingAuthProvider{}

	actual, err := encodedAuth(auth, "ubuntu")
	assert.NilError(t, err)
	expected, err := authconfig.Encode(registrytypes.AuthConfig{Username: "envuser", Password: "envpass", ServerAddress: "https://index.docker.io/v1/"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
	assert.Check(t, is.Len(auth.hosts, 0), "the auth provider must not be asked for a registry in DOCKER_AUTH_CONFIG")
}

func TestEncodedAuthFromConfigFile(t *testing.T) {
	const configFileContent = `{"auths": {
		"https://index.docker.io/v1/": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="},
		"localhost:5000": {"auth": "dXNlcm5hbWU6cGFzc3dvcmQ="}
	}
}`
	cfg := configfile.ConfigFile{}
	assert.NilError(t, cfg.LoadFromReader(bytes.NewReader([]byte(configFileContent))))

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
			actual, err := encodedAuth(&cfg, tc.image)
			assert.NilError(t, err)
			expected, err := authconfig.Encode(tc.expected)
			assert.NilError(t, err)
			assert.Check(t, is.Equal(actual, expected))
		})
	}
}
