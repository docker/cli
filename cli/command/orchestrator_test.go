package command

import (
	"io/ioutil"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/env"
)

func TestOrchestratorSwitch(t *testing.T) {
	var testcases = []struct {
		doc                  string
		globalOrchestrator   string
		envOrchestrator      string
		flagOrchestrator     string
		contextOrchestrator  string
		expectedOrchestrator string
		expectedKubernetes   bool
		expectedSwarm        bool
	}{
		{
			doc:                  "default",
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
		{
			doc:                  "kubernetesConfigFile",
			globalOrchestrator:   "kubernetes",
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
			expectedSwarm:        false,
		},
		{
			doc:                  "kubernetesEnv",
			envOrchestrator:      "kubernetes",
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
			expectedSwarm:        false,
		},
		{
			doc:                  "kubernetesFlag",
			flagOrchestrator:     "kubernetes",
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
			expectedSwarm:        false,
		},
		{
			doc:                  "allOrchestratorFlag",
			flagOrchestrator:     "all",
			expectedOrchestrator: "all",
			expectedKubernetes:   true,
			expectedSwarm:        true,
		},
		{
			doc:                  "kubernetesContext",
			contextOrchestrator:  "kubernetes",
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
		},
		{
			doc:                  "contextOverridesConfigFile",
			globalOrchestrator:   "kubernetes",
			contextOrchestrator:  "swarm",
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
		{
			doc:                  "envOverridesConfigFile",
			globalOrchestrator:   "kubernetes",
			envOrchestrator:      "swarm",
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
		{
			doc:                  "flagOverridesEnv",
			envOrchestrator:      "kubernetes",
			flagOrchestrator:     "swarm",
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			if testcase.envOrchestrator != "" {
				defer env.Patch(t, "DOCKER_STACK_ORCHESTRATOR", testcase.envOrchestrator)()
			}
			orchestrator, err := GetStackOrchestrator(testcase.flagOrchestrator, testcase.contextOrchestrator, testcase.globalOrchestrator, ioutil.Discard)
			assert.NilError(t, err)
			assert.Check(t, is.Equal(testcase.expectedKubernetes, orchestrator.HasKubernetes()))
			assert.Check(t, is.Equal(testcase.expectedSwarm, orchestrator.HasSwarm()))
			assert.Check(t, is.Equal(testcase.expectedOrchestrator, string(orchestrator)))
		})
	}
}
