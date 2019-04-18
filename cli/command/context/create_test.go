package context

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/cli/internal/test"
	"gotest.tools/assert"
	"gotest.tools/env"
)

func makeFakeCli(t *testing.T, opts ...func(*test.FakeCli)) (*test.FakeCli, func()) {
	dir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	storeConfig := store.NewConfig(
		func() interface{} { return &command.DockerContext{} },
		store.EndpointTypeGetter(docker.DockerEndpoint, func() interface{} { return &docker.EndpointMeta{} }),
		store.EndpointTypeGetter(kubernetes.KubernetesEndpoint, func() interface{} { return &kubernetes.EndpointMeta{} }),
	)
	store := &command.ContextStoreWithDefault{
		Store: store.New(dir, storeConfig),
		Resolver: func() (*command.DefaultContext, error) {
			return &command.DefaultContext{
				Meta: store.ContextMetadata{
					Endpoints: map[string]interface{}{
						docker.DockerEndpoint: docker.EndpointMeta{
							Host: "unix:///var/run/docker.sock",
						},
					},
					Metadata: command.DockerContext{
						Description:       "",
						StackOrchestrator: command.OrchestratorSwarm,
					},
					Name: command.DefaultContextName,
				},
				TLS: store.ContextTLSData{},
			}, nil
		},
	}
	cleanup := func() {
		os.RemoveAll(dir)
	}
	result := test.NewFakeCli(nil, opts...)
	for _, o := range opts {
		o(result)
	}
	result.SetContextStore(store)
	return result, cleanup
}

func withCliConfig(configFile *configfile.ConfigFile) func(*test.FakeCli) {
	return func(m *test.FakeCli) {
		m.SetConfigFile(configFile)
	}
}

func TestCreateInvalids(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	assert.NilError(t, cli.ContextStore().CreateOrUpdateContext(store.ContextMetadata{Name: "existing-context"}))
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
		{
			options: CreateOptions{
				Name:                     "invalid-orchestrator",
				DefaultStackOrchestrator: "invalid",
			},
			expecterErr: `specified orchestrator "invalid" is invalid, please use either kubernetes, swarm or all`,
		},
		{
			options: CreateOptions{
				Name:                     "orchestrator-kubernetes-no-endpoint",
				DefaultStackOrchestrator: "kubernetes",
				Docker:                   map[string]string{},
			},
			expecterErr: `cannot specify orchestrator "kubernetes" without configuring a Kubernetes endpoint`,
		},
		{
			options: CreateOptions{
				Name:                     "orchestrator-all-no-endpoint",
				DefaultStackOrchestrator: "all",
				Docker:                   map[string]string{},
			},
			expecterErr: `cannot specify orchestrator "all" without configuring a Kubernetes endpoint`,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.options.Name, func(t *testing.T) {
			err := RunCreate(cli, &tc.options)
			assert.ErrorContains(t, err, tc.expecterErr)
		})
	}
}

func TestCreateOrchestratorSwarm(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()

	err := RunCreate(cli, &CreateOptions{
		Name:                     "test",
		DefaultStackOrchestrator: "swarm",
		Docker:                   map[string]string{},
	})
	assert.NilError(t, err)
	assert.Equal(t, "test\n", cli.OutBuffer().String())
	assert.Equal(t, "Successfully created context \"test\"\n", cli.ErrBuffer().String())
}

func TestCreateOrchestratorEmpty(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()

	err := RunCreate(cli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
	})
	assert.NilError(t, err)
}

func validateTestKubeEndpoint(t *testing.T, s store.Reader, name string) {
	t.Helper()
	ctxMetadata, err := s.GetContextMetadata(name)
	assert.NilError(t, err)
	kubeMeta := ctxMetadata.Endpoints[kubernetes.KubernetesEndpoint].(kubernetes.EndpointMeta)
	kubeEP, err := kubeMeta.WithTLSData(s, name)
	assert.NilError(t, err)
	assert.Equal(t, "https://someserver", kubeEP.Host)
	assert.Equal(t, "the-ca", string(kubeEP.TLSData.CA))
	assert.Equal(t, "the-cert", string(kubeEP.TLSData.Cert))
	assert.Equal(t, "the-key", string(kubeEP.TLSData.Key))
}

func createTestContextWithKube(t *testing.T, cli command.Cli) {
	t.Helper()
	revert := env.Patch(t, "KUBECONFIG", "./testdata/test-kubeconfig")
	defer revert()

	err := RunCreate(cli, &CreateOptions{
		Name:                     "test",
		DefaultStackOrchestrator: "all",
		Kubernetes: map[string]string{
			keyFrom: "default",
		},
		Docker: map[string]string{},
	})
	assert.NilError(t, err)
}

func TestCreateOrchestratorAllKubernetesEndpointFromCurrent(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKube(t, cli)
	validateTestKubeEndpoint(t, cli.ContextStore(), "test")
}

func TestCreateFromContext(t *testing.T) {
	cases := []struct {
		name                 string
		description          string
		orchestrator         string
		expectedDescription  string
		docker               map[string]string
		kubernetes           map[string]string
		expectedOrchestrator command.Orchestrator
	}{
		{
			name:                 "no-override",
			expectedDescription:  "original description",
			expectedOrchestrator: command.OrchestratorSwarm,
		},
		{
			name:                 "override-description",
			description:          "new description",
			expectedDescription:  "new description",
			expectedOrchestrator: command.OrchestratorSwarm,
		},
		{
			name:                 "override-orchestrator",
			orchestrator:         "kubernetes",
			expectedDescription:  "original description",
			expectedOrchestrator: command.OrchestratorKubernetes,
		},
	}

	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	revert := env.Patch(t, "KUBECONFIG", "./testdata/test-kubeconfig")
	defer revert()
	assert.NilError(t, RunCreate(cli, &CreateOptions{
		Name:        "original",
		Description: "original description",
		Docker: map[string]string{
			keyHost: "tcp://42.42.42.42:2375",
		},
		Kubernetes: map[string]string{
			keyFrom: "default",
		},
		DefaultStackOrchestrator: "swarm",
	}))
	assert.NilError(t, RunCreate(cli, &CreateOptions{
		Name:        "dummy",
		Description: "dummy description",
		Docker: map[string]string{
			keyHost: "tcp://24.24.24.24:2375",
		},
		Kubernetes: map[string]string{
			keyFrom: "default",
		},
		DefaultStackOrchestrator: "swarm",
	}))

	cli.SetCurrentContext("dummy")

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := RunCreate(cli, &CreateOptions{
				From:                     "original",
				Name:                     c.name,
				Description:              c.description,
				DefaultStackOrchestrator: c.orchestrator,
				Docker:                   c.docker,
				Kubernetes:               c.kubernetes,
			})
			assert.NilError(t, err)
			newContext, err := cli.ContextStore().GetContextMetadata(c.name)
			assert.NilError(t, err)
			newContextTyped, err := command.GetDockerContext(newContext)
			assert.NilError(t, err)
			dockerEndpoint, err := docker.EndpointFromContext(newContext)
			assert.NilError(t, err)
			kubeEndpoint := kubernetes.EndpointFromContext(newContext)
			assert.Check(t, kubeEndpoint != nil)
			assert.Equal(t, newContextTyped.Description, c.expectedDescription)
			assert.Equal(t, newContextTyped.StackOrchestrator, c.expectedOrchestrator)
			assert.Equal(t, dockerEndpoint.Host, "tcp://42.42.42.42:2375")
			assert.Equal(t, kubeEndpoint.Host, "https://someserver")
		})
	}
}

func TestCreateFromCurrent(t *testing.T) {
	cases := []struct {
		name                 string
		description          string
		orchestrator         string
		expectedDescription  string
		expectedOrchestrator command.Orchestrator
	}{
		{
			name:                 "no-override",
			expectedDescription:  "original description",
			expectedOrchestrator: command.OrchestratorSwarm,
		},
		{
			name:                 "override-description",
			description:          "new description",
			expectedDescription:  "new description",
			expectedOrchestrator: command.OrchestratorSwarm,
		},
		{
			name:                 "override-orchestrator",
			orchestrator:         "kubernetes",
			expectedDescription:  "original description",
			expectedOrchestrator: command.OrchestratorKubernetes,
		},
	}

	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	revert := env.Patch(t, "KUBECONFIG", "./testdata/test-kubeconfig")
	defer revert()
	assert.NilError(t, RunCreate(cli, &CreateOptions{
		Name:        "original",
		Description: "original description",
		Docker: map[string]string{
			keyHost: "tcp://42.42.42.42:2375",
		},
		Kubernetes: map[string]string{
			keyFrom: "default",
		},
		DefaultStackOrchestrator: "swarm",
	}))

	cli.SetCurrentContext("original")

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := RunCreate(cli, &CreateOptions{
				Name:                     c.name,
				Description:              c.description,
				DefaultStackOrchestrator: c.orchestrator,
			})
			assert.NilError(t, err)
			newContext, err := cli.ContextStore().GetContextMetadata(c.name)
			assert.NilError(t, err)
			newContextTyped, err := command.GetDockerContext(newContext)
			assert.NilError(t, err)
			dockerEndpoint, err := docker.EndpointFromContext(newContext)
			assert.NilError(t, err)
			kubeEndpoint := kubernetes.EndpointFromContext(newContext)
			assert.Check(t, kubeEndpoint != nil)
			assert.Equal(t, newContextTyped.Description, c.expectedDescription)
			assert.Equal(t, newContextTyped.StackOrchestrator, c.expectedOrchestrator)
			assert.Equal(t, dockerEndpoint.Host, "tcp://42.42.42.42:2375")
			assert.Equal(t, kubeEndpoint.Host, "https://someserver")
		})
	}
}
