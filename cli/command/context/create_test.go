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
	store := store.New(dir, storeConfig)
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
		options     createOptions
		expecterErr string
	}{
		{
			expecterErr: `context name cannot be empty`,
		},
		{
			options: createOptions{
				name: " ",
			},
			expecterErr: `context name " " is invalid`,
		},
		{
			options: createOptions{
				name: "existing-context",
			},
			expecterErr: `context "existing-context" already exists`,
		},
		{
			options: createOptions{
				name: "invalid-docker-host",
				docker: map[string]string{
					keyHost: "some///invalid/host",
				},
			},
			expecterErr: `unable to parse docker host`,
		},
		{
			options: createOptions{
				name:                     "invalid-orchestrator",
				defaultStackOrchestrator: "invalid",
			},
			expecterErr: `specified orchestrator "invalid" is invalid, please use either kubernetes, swarm or all`,
		},
		{
			options: createOptions{
				name:                     "orchestrator-swarm-no-endpoint",
				defaultStackOrchestrator: "swarm",
			},
			expecterErr: `docker endpoint configuration is required`,
		},
		{
			options: createOptions{
				name:                     "orchestrator-kubernetes-no-endpoint",
				defaultStackOrchestrator: "kubernetes",
				docker:                   map[string]string{},
			},
			expecterErr: `cannot specify orchestrator "kubernetes" without configuring a Kubernetes endpoint`,
		},
		{
			options: createOptions{
				name:                     "orchestrator-all-no-endpoint",
				defaultStackOrchestrator: "all",
				docker:                   map[string]string{},
			},
			expecterErr: `cannot specify orchestrator "all" without configuring a Kubernetes endpoint`,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.options.name, func(t *testing.T) {
			err := runCreate(cli, &tc.options)
			assert.ErrorContains(t, err, tc.expecterErr)
		})
	}
}

func TestCreateOrchestratorSwarm(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()

	err := runCreate(cli, &createOptions{
		name:                     "test",
		defaultStackOrchestrator: "swarm",
		docker:                   map[string]string{},
	})
	assert.NilError(t, err)
	assert.Equal(t, "test\n", cli.OutBuffer().String())
	assert.Equal(t, "Successfully created context \"test\"\n", cli.ErrBuffer().String())
}

func TestCreateOrchestratorEmpty(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()

	err := runCreate(cli, &createOptions{
		name:   "test",
		docker: map[string]string{},
	})
	assert.NilError(t, err)
}

func validateTestKubeEndpoint(t *testing.T, s store.Store, name string) {
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

	err := runCreate(cli, &createOptions{
		name:                     "test",
		defaultStackOrchestrator: "all",
		kubernetes: map[string]string{
			keyFromCurrent: "true",
		},
		docker: map[string]string{},
	})
	assert.NilError(t, err)
}

func TestCreateOrchestratorAllKubernetesEndpointFromCurrent(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKube(t, cli)
	validateTestKubeEndpoint(t, cli.ContextStore(), "test")
}
