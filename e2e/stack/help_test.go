package stack

import (
	"fmt"
	"testing"

	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

func TestStackDeployHelp(t *testing.T) {
	t.Run("Swarm", func(t *testing.T) {
		testStackDeployHelp(t, "swarm")
	})
}

func testStackDeployHelp(t *testing.T, orchestrator string) {
	result := icmd.RunCommand("docker", "stack", "deploy", "--orchestrator", orchestrator, "--help")
	result.Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), fmt.Sprintf("stack-deploy-help-%s.golden", orchestrator))
}
