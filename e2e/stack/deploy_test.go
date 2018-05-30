package stack

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test/environment"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/gotestyourself/gotestyourself/skip"
)

func TestDeployWithNamedResources(t *testing.T) {
	t.Run("Swarm", func(t *testing.T) {
		testDeployWithNamedResources(t, "swarm")
	})
	t.Run("Kubernetes", func(t *testing.T) {
		// FIXME(chris-crone): currently does not work with compose for kubernetes.
		t.Skip("FIXME(chris-crone): currently does not work with compose for kubernetes.")
		skip.If(t, !environment.KubernetesEnabled())

		testDeployWithNamedResources(t, "kubernetes")
	})
}

func testDeployWithNamedResources(t *testing.T, orchestrator string) {
	stackname := fmt.Sprintf("test-stack-deploy-with-names-%s", orchestrator)
	composefile := golden.Path("stack-with-named-resources.yml")

	result := icmd.RunCommand("docker", "--orchestrator", orchestrator,
		"stack", "deploy", "-c", composefile, stackname)
	defer icmd.RunCommand("docker", "--orchestrator", orchestrator,
		"stack", "rm", stackname)

	result.Assert(t, icmd.Success)
	stdout := strings.Split(result.Stdout(), "\n")
	expected := strings.Split(string(golden.Get(t, fmt.Sprintf("stack-deploy-with-names-%s.golden", orchestrator))), "\n")
	sort.Strings(stdout)
	sort.Strings(expected)
	assert.DeepEqual(t, stdout, expected)
}
