package stack

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

func TestDeployWithNamedResources(t *testing.T) {
	t.Run("Swarm", func(t *testing.T) {
		testDeployWithNamedResources(t, "swarm")
	})
}

func testDeployWithNamedResources(t *testing.T, orchestrator string) {
	stackname := fmt.Sprintf("test-stack-deploy-with-names-%s", orchestrator)
	composefile := golden.Path("stack-with-named-resources.yml")

	result := icmd.RunCommand("docker", "stack", "deploy",
		"-c", composefile, stackname, "--orchestrator", orchestrator)
	defer icmd.RunCommand("docker", "stack", "rm",
		"--orchestrator", orchestrator, stackname)

	result.Assert(t, icmd.Success)
	stdout := strings.Split(result.Stdout(), "\n")
	expected := strings.Split(string(golden.Get(t, fmt.Sprintf("stack-deploy-with-names-%s.golden", orchestrator))), "\n")
	sort.Strings(stdout)
	sort.Strings(expected)
	assert.DeepEqual(t, stdout, expected)
}
