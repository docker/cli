// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

package context

import (
	"fmt"
	"testing"

	"github.com/docker/cli/cli/command/internal/cli"
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
		func() any { return &cli.DockerContext{} },
		store.EndpointTypeGetter(docker.DockerEndpoint, func() any { return &docker.EndpointMeta{} }),
	)
	contextStore := &cli.ContextStoreWithDefault{
		Store: store.New(dir, storeConfig),
		Resolver: func() (*cli.DefaultContext, error) {
			return &cli.DefaultContext{
				Meta: store.Metadata{
					Endpoints: map[string]any{
						docker.DockerEndpoint: docker.EndpointMeta{
							Host: "unix:///var/run/docker.sock",
						},
					},
					Metadata: cli.DockerContext{
						Description: "",
					},
					Name: cli.DefaultContextName,
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
	fakeCli := makeFakeCli(t)
	assert.NilError(t, fakeCli.ContextStore().CreateOrUpdate(store.Metadata{Name: "existing-context"}))
	tests := []struct {
		doc         string
		options     CreateOptions
		expecterErr string
	}{
		{
			doc:         "empty name",
			expecterErr: `context name cannot be empty`,
		},
		{
			doc: "reserved name",
			options: CreateOptions{
				Name: "default",
			},
			expecterErr: `"default" is a reserved context name`,
		},
		{
			doc: "whitespace-only name",
			options: CreateOptions{
				Name: " ",
			},
			expecterErr: `context name " " is invalid`,
		},
		{
			doc: "existing context",
			options: CreateOptions{
				Name: "existing-context",
			},
			expecterErr: `context "existing-context" already exists`,
		},
		{
			doc: "invalid docker host",
			options: CreateOptions{
				Name: "invalid-docker-host",
				Docker: map[string]string{
					"host": "some///invalid/host",
				},
			},
			expecterErr: `unable to parse docker host`,
		},
		{
			doc: "ssh host with skip-tls-verify=false",
			options: CreateOptions{
				Name: "skip-tls-verify-false",
				Docker: map[string]string{
					"host": "ssh://example.com,skip-tls-verify=false",
				},
			},
		},
		{
			doc: "ssh host with skip-tls-verify=true",
			options: CreateOptions{
				Name: "skip-tls-verify-true",
				Docker: map[string]string{
					"host": "ssh://example.com,skip-tls-verify=true",
				},
			},
		},
		{
			doc: "ssh host with skip-tls-verify=INVALID",
			options: CreateOptions{
				Name: "skip-tls-verify-invalid",
				Docker: map[string]string{
					"host":            "ssh://example.com",
					"skip-tls-verify": "INVALID",
				},
			},
			expecterErr: `unable to create docker endpoint config: skip-tls-verify: parsing "INVALID": invalid syntax`,
		},
		{
			doc: "unknown option",
			options: CreateOptions{
				Name: "unknown-option",
				Docker: map[string]string{
					"UNKNOWN": "value",
				},
			},
			expecterErr: `unable to create docker endpoint config: unrecognized config key: UNKNOWN`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			err := RunCreate(fakeCli, &tc.options)
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

	err := RunCreate(cli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
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

	fakeCli := makeFakeCli(t)
	fakeCli.ResetOutputBuffers()
	assert.NilError(t, RunCreate(fakeCli, &CreateOptions{
		Name:        "original",
		Description: "original description",
		Docker: map[string]string{
			keyHost: "tcp://42.42.42.42:2375",
		},
	}))
	assertContextCreateLogging(t, fakeCli, "original")

	fakeCli.ResetOutputBuffers()
	assert.NilError(t, RunCreate(fakeCli, &CreateOptions{
		Name:        "dummy",
		Description: "dummy description",
		Docker: map[string]string{
			keyHost: "tcp://24.24.24.24:2375",
		},
	}))
	assertContextCreateLogging(t, fakeCli, "dummy")

	fakeCli.SetCurrentContext("dummy")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeCli.ResetOutputBuffers()
			err := RunCreate(fakeCli, &CreateOptions{
				From:        "original",
				Name:        tc.name,
				Description: tc.description,
				Docker:      tc.docker,
			})
			assert.NilError(t, err)
			assertContextCreateLogging(t, fakeCli, tc.name)
			newContext, err := fakeCli.ContextStore().GetMetadata(tc.name)
			assert.NilError(t, err)
			newContextTyped, err := cli.GetDockerContext(newContext)
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

	fakeCli := makeFakeCli(t)
	fakeCli.ResetOutputBuffers()
	assert.NilError(t, RunCreate(fakeCli, &CreateOptions{
		Name:        "original",
		Description: "original description",
		Docker: map[string]string{
			keyHost: "tcp://42.42.42.42:2375",
		},
	}))
	assertContextCreateLogging(t, fakeCli, "original")

	fakeCli.SetCurrentContext("original")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeCli.ResetOutputBuffers()
			err := RunCreate(fakeCli, &CreateOptions{
				Name:        tc.name,
				Description: tc.description,
			})
			assert.NilError(t, err)
			assertContextCreateLogging(t, fakeCli, tc.name)
			newContext, err := fakeCli.ContextStore().GetMetadata(tc.name)
			assert.NilError(t, err)
			newContextTyped, err := cli.GetDockerContext(newContext)
			assert.NilError(t, err)
			dockerEndpoint, err := docker.EndpointFromContext(newContext)
			assert.NilError(t, err)
			assert.Equal(t, newContextTyped.Description, tc.expectedDescription)
			assert.Equal(t, dockerEndpoint.Host, "tcp://42.42.42.42:2375")
		})
	}
}
