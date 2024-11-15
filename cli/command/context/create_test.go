// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

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
		options     CreateOptions
		expecterErr string
	}{
		{
			expecterErr: `context name cannot be empty`,
		},
		{
			options: CreateOptions{
				Name: "default",
			},
			expecterErr: `"default" is a reserved context name`,
		},
		{
			options: CreateOptions{
				Name: " ",
			},
			expecterErr: `context name " " is invalid`,
		},
		{
			options: CreateOptions{
				Name: "existing-context",
			},
			expecterErr: `context "existing-context" already exists`,
		},
		{
			options: CreateOptions{
				Name: "invalid-docker-host",
				Docker: map[string]string{
					keyHost: "some///invalid/host",
				},
			},
			expecterErr: `unable to parse docker host`,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.options.Name, func(t *testing.T) {
			err := RunCreate(cli, &tc.options)
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

	cli := makeFakeCli(t)
	cli.ResetOutputBuffers()
	assert.NilError(t, RunCreate(cli, &CreateOptions{
		Name:        "original",
		Description: "original description",
		Docker: map[string]string{
			keyHost: "tcp://42.42.42.42:2375",
		},
	}))
	assertContextCreateLogging(t, cli, "original")

	cli.ResetOutputBuffers()
	assert.NilError(t, RunCreate(cli, &CreateOptions{
		Name:        "dummy",
		Description: "dummy description",
		Docker: map[string]string{
			keyHost: "tcp://24.24.24.24:2375",
		},
	}))
	assertContextCreateLogging(t, cli, "dummy")

	cli.SetCurrentContext("dummy")

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			cli.ResetOutputBuffers()
			err := RunCreate(cli, &CreateOptions{
				From:        "original",
				Name:        c.name,
				Description: c.description,
				Docker:      c.docker,
			})
			assert.NilError(t, err)
			assertContextCreateLogging(t, cli, c.name)
			newContext, err := cli.ContextStore().GetMetadata(c.name)
			assert.NilError(t, err)
			newContextTyped, err := command.GetDockerContext(newContext)
			assert.NilError(t, err)
			dockerEndpoint, err := docker.EndpointFromContext(newContext)
			assert.NilError(t, err)
			assert.Equal(t, newContextTyped.Description, c.expectedDescription)
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
	assert.NilError(t, RunCreate(cli, &CreateOptions{
		Name:        "original",
		Description: "original description",
		Docker: map[string]string{
			keyHost: "tcp://42.42.42.42:2375",
		},
	}))
	assertContextCreateLogging(t, cli, "original")

	cli.SetCurrentContext("original")

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			cli.ResetOutputBuffers()
			err := RunCreate(cli, &CreateOptions{
				Name:        c.name,
				Description: c.description,
			})
			assert.NilError(t, err)
			assertContextCreateLogging(t, cli, c.name)
			newContext, err := cli.ContextStore().GetMetadata(c.name)
			assert.NilError(t, err)
			newContextTyped, err := command.GetDockerContext(newContext)
			assert.NilError(t, err)
			dockerEndpoint, err := docker.EndpointFromContext(newContext)
			assert.NilError(t, err)
			assert.Equal(t, newContextTyped.Description, c.expectedDescription)
			assert.Equal(t, dockerEndpoint.Host, "tcp://42.42.42.42:2375")
		})
	}
}
