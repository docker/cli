// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package context

import (
	"fmt"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func makeFakeCli(t *testing.T, opts ...func(*test.FakeCli)) *test.FakeCli {
	t.Helper()
	dir := t.TempDir()
	storeConfig := store.NewConfig(
		func() any { return &command.DockerContext{} },
		store.EndpointTypeGetter(docker.DockerEndpoint, func() any { return &docker.EndpointMeta{} }),
	)
	contextStore := &command.ContextStoreWithDefault{
		Store: store.New(dir, storeConfig),
		Resolver: func() (*command.DefaultContext, error) {
			return &command.DefaultContext{
				Meta: store.Metadata{
					Endpoints: map[string]any{
						docker.DockerEndpoint: docker.EndpointMeta{
							Host: "unix:///var/run/docker.sock",
						},
					},
					Metadata: command.DockerContext{
						Description: "",
					},
					Name: command.DefaultContextName,
				},
				TLS: store.ContextTLSData{},
			}, nil
		},
	}
	result := test.NewFakeCli(nil, opts...)
	for _, o := range opts {
		o(result)
	}
	result.SetContextStore(contextStore)
	return result
}

func withCliConfig(configFile *configfile.ConfigFile) func(*test.FakeCli) {
	return func(m *test.FakeCli) {
		m.SetConfigFile(configFile)
	}
}

func TestCreate(t *testing.T) {
	cli := makeFakeCli(t)
	assert.NilError(t, cli.ContextStore().CreateOrUpdate(store.Metadata{Name: "existing-context"}))
	tests := []struct {
		doc         string
		options     createOptions
		name        string
		expecterErr string
	}{
		{
			doc:         "empty name",
			expecterErr: `context name cannot be empty`,
		},
		{
			doc:         "reserved name",
			name:        "default",
			expecterErr: `"default" is a reserved context name`,
		},
		{
			doc:         "whitespace-only name",
			name:        " ",
			expecterErr: `context name " " is invalid`,
		},
		{
			doc:         "existing context",
			name:        "existing-context",
			expecterErr: `context "existing-context" already exists`,
		},
		{
			doc:  "invalid docker host",
			name: "invalid-docker-host",
			options: createOptions{
				endpoint: map[string]string{
					"host": "some///invalid/host",
				},
			},
			expecterErr: `unable to parse docker host`,
		},
		{
			doc:  "ssh host with skip-tls-verify=false",
			name: "skip-tls-verify-false",
			options: createOptions{
				endpoint: map[string]string{
					"host": "ssh://example.com,skip-tls-verify=false",
				},
			},
		},
		{
			doc:  "ssh host with skip-tls-verify=true",
			name: "skip-tls-verify-true",
			options: createOptions{
				endpoint: map[string]string{
					"host": "ssh://example.com,skip-tls-verify=true",
				},
			},
		},
		{
			doc:  "ssh host with skip-tls-verify=INVALID",
			name: "skip-tls-verify-invalid",
			options: createOptions{
				endpoint: map[string]string{
					"host":            "ssh://example.com",
					"skip-tls-verify": "INVALID",
				},
			},
			expecterErr: `unable to create docker endpoint config: skip-tls-verify: parsing "INVALID": invalid syntax`,
		},
		{
			doc:  "unknown option",
			name: "unknown-option",
			options: createOptions{
				endpoint: map[string]string{
					"UNKNOWN": "value",
				},
			},
			expecterErr: `unable to create docker endpoint config: unrecognized config key: UNKNOWN`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			err := runCreate(cli, tc.name, tc.options)
			if tc.expecterErr == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expecterErr)
			}
		})
	}
}

func assertContextCreateLogging(t *testing.T, cli *test.FakeCli, n string) {
	t.Helper()
	assert.Equal(t, n+"\n", cli.OutBuffer().String())
	assert.Equal(t, fmt.Sprintf("Successfully created context %q\n", n), cli.ErrBuffer().String())
}

func TestCreateOrchestratorEmpty(t *testing.T) {
	cli := makeFakeCli(t)

	err := runCreate(cli, "test", createOptions{
		endpoint: map[string]string{},
	})
	assert.NilError(t, err)
	assertContextCreateLogging(t, cli, "test")
}

func TestCreateFromContext(t *testing.T) {
	cases := []struct {
		name                string
		description         string
		expectedDescription string
		docker              map[string]string
	}{
		{
			name:                "no-override",
			expectedDescription: "original description",
		},
		{
			name:                "override-description",
			description:         "new description",
			expectedDescription: "new description",
		},
	}

	cli := makeFakeCli(t)
	cli.ResetOutputBuffers()
	assert.NilError(t, runCreate(cli, "original", createOptions{
		description: "original description",
		endpoint: map[string]string{
			keyHost: "tcp://42.42.42.42:2375",
		},
	}))
	assertContextCreateLogging(t, cli, "original")

	cli.ResetOutputBuffers()
	assert.NilError(t, runCreate(cli, "dummy", createOptions{
		description: "dummy description",
		endpoint: map[string]string{
			keyHost: "tcp://24.24.24.24:2375",
		},
	}))
	assertContextCreateLogging(t, cli, "dummy")

	cli.SetCurrentContext("dummy")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cli.ResetOutputBuffers()
			err := runCreate(cli, tc.name, createOptions{
				from:        "original",
				description: tc.description,
				endpoint:    tc.docker,
			})
			assert.NilError(t, err)
			assertContextCreateLogging(t, cli, tc.name)
			newContext, err := cli.ContextStore().GetMetadata(tc.name)
			assert.NilError(t, err)
			newContextTyped, err := command.GetDockerContext(newContext)
			assert.NilError(t, err)
			dockerEndpoint, err := docker.EndpointFromContext(newContext)
			assert.NilError(t, err)
			assert.Equal(t, newContextTyped.Description, tc.expectedDescription)
			assert.Equal(t, dockerEndpoint.Host, "tcp://42.42.42.42:2375")
		})
	}
}

func TestCreateFromCurrent(t *testing.T) {
	cases := []struct {
		name                string
		description         string
		orchestrator        string
		expectedDescription string
	}{
		{
			name:                "no-override",
			expectedDescription: "original description",
		},
		{
			name:                "override-description",
			description:         "new description",
			expectedDescription: "new description",
		},
	}

	cli := makeFakeCli(t)
	cli.ResetOutputBuffers()
	assert.NilError(t, runCreate(cli, "original", createOptions{
		description: "original description",
		endpoint: map[string]string{
			keyHost: "tcp://42.42.42.42:2375",
		},
	}))
	assertContextCreateLogging(t, cli, "original")

	cli.SetCurrentContext("original")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cli.ResetOutputBuffers()
			err := runCreate(cli, tc.name, createOptions{
				description: tc.description,
			})
			assert.NilError(t, err)
			assertContextCreateLogging(t, cli, tc.name)
			newContext, err := cli.ContextStore().GetMetadata(tc.name)
			assert.NilError(t, err)
			newContextTyped, err := command.GetDockerContext(newContext)
			assert.NilError(t, err)
			dockerEndpoint, err := docker.EndpointFromContext(newContext)
			assert.NilError(t, err)
			assert.Equal(t, newContextTyped.Description, tc.expectedDescription)
			assert.Equal(t, dockerEndpoint.Host, "tcp://42.42.42.42:2375")
		})
	}
}
