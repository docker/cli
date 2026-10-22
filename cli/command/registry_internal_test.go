package command

import (
	"bytes"
	"path"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
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

func TestRetrieveAuthTokenFromImageUsesAuthProvider(t *testing.T) {
	auth := &recordingAuthProvider{}

	actual, err := retrieveAuthTokenFromImage(auth, "dhi.io/python")
	assert.NilError(t, err)
	expected, err := authconfig.Encode(registrytypes.AuthConfig{Username: "hubuser", Password: "desktop-token", ServerAddress: "dhi.io"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
	assert.Check(t, is.DeepEqual(auth.hosts, []string{"dhi.io"}))
}

func TestRetrieveAuthTokenFromImageEnvTakesPrecedence(t *testing.T) {
	t.Setenv(configfile.DockerEnvConfigKey, `{"auths":{"https://index.docker.io/v1/":{"auth":"ZW52dXNlcjplbnZwYXNz"}}}`)
	auth := &recordingAuthProvider{}

	actual, err := retrieveAuthTokenFromImage(auth, "ubuntu")
	assert.NilError(t, err)
	expected, err := authconfig.Encode(registrytypes.AuthConfig{Username: "envuser", Password: "envpass", ServerAddress: "https://index.docker.io/v1/"})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(actual, expected))
	assert.Check(t, is.Len(auth.hosts, 0), "the auth provider must not be asked for a registry in DOCKER_AUTH_CONFIG")
}

func TestRetrieveAuthTokenFromImage(t *testing.T) {
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
		expectedAuthCfg registrytypes.AuthConfig
	}{
		{
			prefix:          "",
			expectedAddress: "https://index.docker.io/v1/",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "https://index.docker.io/v1/"},
		},
		{
			prefix:          "docker.io",
			expectedAddress: "https://index.docker.io/v1/",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "https://index.docker.io/v1/"},
		},
		{
			prefix:          "index.docker.io",
			expectedAddress: "https://index.docker.io/v1/",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "https://index.docker.io/v1/"},
		},
		{
			// FIXME(thaJeztah): registry-1.docker.io (the actual registry) is the odd one out, and is stored separate from other URLs used for docker hub's registry
			prefix:          "registry-1.docker.io",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "registry-1.docker.io"},
		},
		{
			// FIXME(thaJeztah): registry.hub.docker.com is stored separate from other URLs used for docker hub's registry
			prefix:          "registry.hub.docker.com",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "registry.hub.docker.com"},
		},
		{
			prefix:          "[::1]",
			expectedAddress: "[::1]",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "[::1]"},
		},
		{
			prefix:          "[::1]:5000",
			expectedAddress: "[::1]:5000",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "[::1]:5000"},
		},
		{
			prefix:          "127.0.0.1",
			expectedAddress: "127.0.0.1",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "127.0.0.1"},
		},
		{
			prefix:          "localhost",
			expectedAddress: "localhost",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "localhost"},
		},
		{
			prefix:          "localhost:5000",
			expectedAddress: "localhost:5000",
			expectedAuthCfg: registrytypes.AuthConfig{Username: "username", Password: "password", ServerAddress: "localhost:5000"},
		},
		{
			prefix:          "no-auth.example.com",
			expectedAuthCfg: registrytypes.AuthConfig{},
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
				actual, err := retrieveAuthTokenFromImage(&cfg, imageRef)
				assert.NilError(t, err)
				expectedAuthCfg, err := authconfig.Encode(tc.expectedAuthCfg)
				assert.NilError(t, err)
				assert.Equal(t, actual, expectedAuthCfg)
			}
		})
	}
}
