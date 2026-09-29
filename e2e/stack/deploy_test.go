package stack

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

func TestDeployWithNamedResources(t *testing.T) {
	stackname := "test-stack-deploy-with-names"
	composefile := golden.Path("stack-with-named-resources.yml")

	result := icmd.RunCommand("docker", "stack", "deploy",
		"-c", composefile, stackname)
	defer icmd.RunCommand("docker", "stack", "rm", stackname)

	result.Assert(t, icmd.Success)
	stdout := strings.Split(result.Stdout(), "\n")
	expected := strings.Split(string(golden.Get(t, "stack-deploy-with-names.golden")), "\n")
	assert.DeepEqual(t, stdout, expected, cmpopts.SortSlices(func(a, b string) bool {
		return a < b
	}))
}
