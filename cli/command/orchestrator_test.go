package command

import (
	"io/ioutil"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/env"
)

func TestOrchestratorSwitch(t *testing.T) {
	var testcases = []struct {
		doc                  string
		globalOrchestrator   string
		envOrchestrator      string
		flagOrchestrator     string
		contextOrchestrator  string
		expectedOrchestrator string
		expectedSwarm        bool
	}{
		{
			doc:                  "default",
			expectedOrchestrator: "swarm",
			expectedSwarm:        true,
		},
		{
			doc:                  "allOrchestratorFlag",
			flagOrchestrator:     "all",
			expectedOrchestrator: "all",
			expectedSwarm:        true,
		},
		{
			doc:                  "contextOverridesConfigFile",
			globalOrchestrator:   "kubernetes",
			contextOrchestrator:  "swarm",
			expectedOrchestrator: "swarm",
			expectedSwarm:        true,
		},
		{
			doc:                  "envOverridesConfigFile",
			globalOrchestrator:   "kubernetes",
			envOrchestrator:      "swarm",
			expectedOrchestrator: "swarm",
			expectedSwarm:        true,
		},
		{
			doc:                  "flagOverridesEnv",
			envOrchestrator:      "kubernetes",
			flagOrchestrator:     "swarm",
			expectedOrchestrator: "swarm",
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
			assert.Check(t, is.Equal(testcase.expectedSwarm, orchestrator.HasSwarm()))
			assert.Check(t, is.Equal(testcase.expectedOrchestrator, string(orchestrator)))
		})
	}
}
