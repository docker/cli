package stack

import (
	"fmt"
	"testing"

	"gotest.tools/golden"
	"gotest.tools/icmd"
)

func TestStackDeployHelp(t *testing.T) {
	t.Run("Swarm", func(t *testing.T) {
		testStackDeployHelp(t, "swarm")
	})
	t.Run("Kubernetes", func(t *testing.T) {
		testStackDeployHelp(t, "kubernetes")
	})
}

func testStackDeployHelp(t *testing.T, orchestrator string) {
	result := icmd.RunCommand("docker", "stack", "deploy", "--orchestrator", orchestrator, "--help")
	result.Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), fmt.Sprintf("stack-deploy-help-%s.golden", orchestrator))
}
