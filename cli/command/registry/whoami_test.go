package registry

import (
	"context"
	"testing"

	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/registry"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"
)

func TestWhoamiNotLoggedIn(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-not-logged-in")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	cli.ConfigFile().Filename = tmpFile.Path()

	err := runWhoami(context.Background(), cli, whoamiOptions{})
	assert.Error(t, err, "not logged in to Docker Hub")
	assert.Check(t, is.Equal("", cli.OutBuffer().String()))
}

func TestWhoamiLoggedInDockerHub(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-docker-hub")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	configfile := cli.ConfigFile()
	configfile.Filename = tmpFile.Path()

	assert.NilError(t, configfile.GetCredentialsStore(registry.IndexServer).Store(configtypes.AuthConfig{
		Username:      "testuser",
		Password:      "testpass",
		ServerAddress: registry.IndexServer,
	}))

	err := runWhoami(context.Background(), cli, whoamiOptions{})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("testuser\n", cli.OutBuffer().String()))
}

func TestWhoamiLoggedInCustomRegistry(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-custom-registry")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	configfile := cli.ConfigFile()
	configfile.Filename = tmpFile.Path()

	customRegistry := "custom.registry.com"
	assert.NilError(t, configfile.GetCredentialsStore(customRegistry).Store(configtypes.AuthConfig{
		Username:      "customuser",
		Password:      "custompass",
		ServerAddress: customRegistry,
	}))

	err := runWhoami(context.Background(), cli, whoamiOptions{
		serverAddress: customRegistry,
	})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("customuser\n", cli.OutBuffer().String()))
}

func TestWhoamiNotLoggedInCustomRegistry(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-not-logged-in-custom")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	cli.ConfigFile().Filename = tmpFile.Path()

	customRegistry := "custom.registry.com"
	err := runWhoami(context.Background(), cli, whoamiOptions{
		serverAddress: customRegistry,
	})
	assert.Error(t, err, "not logged in to "+customRegistry)
}

func TestWhoamiAll(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-all")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	configfile := cli.ConfigFile()
	configfile.Filename = tmpFile.Path()

	assert.NilError(t, configfile.GetCredentialsStore(registry.IndexServer).Store(configtypes.AuthConfig{
		Username:      "hubuser",
		Password:      "hubpass",
		ServerAddress: registry.IndexServer,
	}))

	assert.NilError(t, configfile.GetCredentialsStore("custom1.registry.com").Store(configtypes.AuthConfig{
		Username:      "custom1user",
		Password:      "custom1pass",
		ServerAddress: "custom1.registry.com",
	}))

	assert.NilError(t, configfile.GetCredentialsStore("custom2.registry.com").Store(configtypes.AuthConfig{
		Username:      "custom2user",
		Password:      "custom2pass",
		ServerAddress: "custom2.registry.com",
	}))

	err := runWhoami(context.Background(), cli, whoamiOptions{
		all: true,
	})
	assert.NilError(t, err)

	output := cli.OutBuffer().String()
	assert.Check(t, is.Contains(output, "custom1.registry.com: custom1user"))
	assert.Check(t, is.Contains(output, "custom2.registry.com: custom2user"))
	assert.Check(t, is.Contains(output, registry.IndexServer+": hubuser"))
}

func TestWhoamiAllNotLoggedIn(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-all-not-logged-in")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	cli.ConfigFile().Filename = tmpFile.Path()

	err := runWhoami(context.Background(), cli, whoamiOptions{
		all: true,
	})
	assert.Error(t, err, "not logged in to any registries")
}

func TestWhoamiWithDockerAuthConfig(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-docker-auth-config")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	configfile := cli.ConfigFile()
	configfile.Filename = tmpFile.Path()

	// Store credentials normally
	assert.NilError(t, configfile.GetCredentialsStore(registry.IndexServer).Store(configtypes.AuthConfig{
		Username:      "testuser",
		Password:      "testpass",
		ServerAddress: registry.IndexServer,
	}))

	// Set DOCKER_AUTH_CONFIG environment variable to trigger warning
	t.Setenv("DOCKER_AUTH_CONFIG", `{"auths":{}}`)

	err := runWhoami(context.Background(), cli, whoamiOptions{})
	assert.NilError(t, err)

	// Should print warning about DOCKER_AUTH_CONFIG
	assert.Check(t, is.Contains(cli.ErrBuffer().String(), "DOCKER_AUTH_CONFIG"))
	assert.Check(t, is.Equal("testuser\n", cli.OutBuffer().String()))
}

func TestWhoamiRegistryWithProtocol(t *testing.T) {
	testCases := []struct {
		name     string
		registry string
	}{
		{
			name:     "with https protocol",
			registry: "https://custom.registry.com",
		},
		{
			name:     "with http protocol",
			registry: "http://custom.registry.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile := fs.NewFile(t, "test-whoami-with-protocol")
			defer tmpFile.Remove()

			cli := test.NewFakeCli(&fakeClient{})
			configfile := cli.ConfigFile()
			configfile.Filename = tmpFile.Path()

			// Store with normalized hostname (without protocol)
			assert.NilError(t, configfile.GetCredentialsStore("custom.registry.com").Store(configtypes.AuthConfig{
				Username:      "protocoluser",
				Password:      "protocolpass",
				ServerAddress: "custom.registry.com",
			}))

			// Query with protocol - should still work
			err := runWhoami(context.Background(), cli, whoamiOptions{
				serverAddress: tc.registry,
			})
			assert.NilError(t, err)
			assert.Check(t, is.Equal("protocoluser\n", cli.OutBuffer().String()))
		})
	}
}

func TestWhoamiDockerIOAlias(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-docker-io-alias")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	configfile := cli.ConfigFile()
	configfile.Filename = tmpFile.Path()

	// Store with Docker Hub
	assert.NilError(t, configfile.GetCredentialsStore(registry.IndexServer).Store(configtypes.AuthConfig{
		Username:      "dockeriouser",
		Password:      "dockeriopass",
		ServerAddress: registry.IndexServer,
	}))

	// Query with docker.io should map to Docker Hub
	err := runWhoami(context.Background(), cli, whoamiOptions{
		serverAddress: "docker.io",
	})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("dockeriouser\n", cli.OutBuffer().String()))
}

func TestWhoamiEmptyUsername(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-empty-username")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	configfile := cli.ConfigFile()
	configfile.Filename = tmpFile.Path()

	// Store credentials with empty username (token-based auth)
	assert.NilError(t, configfile.GetCredentialsStore(registry.IndexServer).Store(configtypes.AuthConfig{
		Username:      "",
		IdentityToken: "sometoken",
		ServerAddress: registry.IndexServer,
	}))

	err := runWhoami(context.Background(), cli, whoamiOptions{})
	assert.Error(t, err, "not logged in to Docker Hub")
}

func TestWhoamiAllSkipsEmptyUsernames(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-all-skip-empty")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	configfile := cli.ConfigFile()
	configfile.Filename = tmpFile.Path()

	// Store one with username
	assert.NilError(t, configfile.GetCredentialsStore("custom.registry.com").Store(configtypes.AuthConfig{
		Username:      "customuser",
		Password:      "custompass",
		ServerAddress: "custom.registry.com",
	}))

	// Store one without username (token-based)
	assert.NilError(t, configfile.GetCredentialsStore("token.registry.com").Store(configtypes.AuthConfig{
		Username:      "",
		IdentityToken: "sometoken",
		ServerAddress: "token.registry.com",
	}))

	err := runWhoami(context.Background(), cli, whoamiOptions{
		all: true,
	})
	assert.NilError(t, err)

	output := cli.OutBuffer().String()
	// Should include the one with username
	assert.Check(t, is.Contains(output, "custom.registry.com: customuser"))
	// Should not include the one without username
	assert.Assert(t, !is.Contains(output, "token.registry.com")().Success())
}

func TestWhoamiWithRegistryPort(t *testing.T) {
	tmpFile := fs.NewFile(t, "test-whoami-with-port")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	configfile := cli.ConfigFile()
	configfile.Filename = tmpFile.Path()

	registryWithPort := "custom.registry.com:5000"
	assert.NilError(t, configfile.GetCredentialsStore(registryWithPort).Store(configtypes.AuthConfig{
		Username:      "portuser",
		Password:      "portpass",
		ServerAddress: registryWithPort,
	}))

	err := runWhoami(context.Background(), cli, whoamiOptions{
		serverAddress: registryWithPort,
	})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("portuser\n", cli.OutBuffer().String()))
}

func TestWhoamiClearsEnvironmentVariable(t *testing.T) {
	// Test should not be affected by environment variable
	t.Setenv("DOCKER_AUTH_CONFIG", "")

	tmpFile := fs.NewFile(t, "test-whoami-no-env")
	defer tmpFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	configfile := cli.ConfigFile()
	configfile.Filename = tmpFile.Path()

	assert.NilError(t, configfile.GetCredentialsStore(registry.IndexServer).Store(configtypes.AuthConfig{
		Username:      "fileuser",
		Password:      "filepass",
		ServerAddress: registry.IndexServer,
	}))

	err := runWhoami(context.Background(), cli, whoamiOptions{})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("fileuser\n", cli.OutBuffer().String()))
	// Should not print warning when env var is not set
	assert.Check(t, is.Equal("", cli.ErrBuffer().String()))
}
