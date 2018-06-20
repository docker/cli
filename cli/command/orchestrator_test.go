package command

import (
	"os"
	"testing"

	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/env"
	"gotest.tools/fs"
)

func TestOrchestratorSwitch(t *testing.T) {
	defaultVersion := "v0.00"

	var testcases = []struct {
		doc                  string
		configfile           string
		envOrchestrator      string
		flagOrchestrator     string
		expectedOrchestrator string
		expectedKubernetes   bool
		expectedSwarm        bool
	}{
		{
			doc: "default",
			configfile: `{
			}`,
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
		{
			doc: "kubernetesConfigFile",
			configfile: `{
				"stackOrchestrator": "kubernetes"
			}`,
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
			expectedSwarm:        false,
		},
		{
			doc: "kubernetesEnv",
			configfile: `{
			}`,
			envOrchestrator:      "kubernetes",
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
			expectedSwarm:        false,
		},
		{
			doc: "kubernetesFlag",
			configfile: `{
			}`,
			flagOrchestrator:     "kubernetes",
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
			expectedSwarm:        false,
		},
		{
			doc: "allOrchestratorFlag",
			configfile: `{
			}`,
			flagOrchestrator:     "all",
			expectedOrchestrator: "all",
			expectedKubernetes:   true,
			expectedSwarm:        true,
		},
		{
			doc: "envOverridesConfigFile",
			configfile: `{
				"stackOrchestrator": "kubernetes"
			}`,
			envOrchestrator:      "swarm",
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
		{
			doc: "flagOverridesEnv",
			configfile: `{
			}`,
			envOrchestrator:      "kubernetes",
			flagOrchestrator:     "swarm",
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			dir := fs.NewDir(t, testcase.doc, fs.WithFile("config.json", testcase.configfile))
			defer dir.Remove()
			apiclient := &fakeClient{
				version: defaultVersion,
			}
			if testcase.envOrchestrator != "" {
				defer env.Patch(t, "DOCKER_STACK_ORCHESTRATOR", testcase.envOrchestrator)()
			}

			cli := &DockerCli{client: apiclient, err: os.Stderr}
			cliconfig.SetDir(dir.Path())
			options := flags.NewClientOptions()
			err := cli.Initialize(options)
			assert.NilError(t, err)

			orchestrator, err := GetStackOrchestrator(testcase.flagOrchestrator, cli.ConfigFile().StackOrchestrator)
			assert.NilError(t, err)
			assert.Check(t, is.Equal(testcase.expectedKubernetes, orchestrator.HasKubernetes()))
			assert.Check(t, is.Equal(testcase.expectedSwarm, orchestrator.HasSwarm()))
			assert.Check(t, is.Equal(testcase.expectedOrchestrator, string(orchestrator)))
		})
	}
}
