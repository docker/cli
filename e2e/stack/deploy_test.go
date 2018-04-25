package stack

import (
	"sort"
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
)

func TestDeployWithNamedResources(t *testing.T) {
	stackname := "test-stack-deploy-with-names"
	composefile := golden.Path("stack-with-named-resources.yml")

	result := icmd.RunCommand(
		"docker", "stack", "deploy", "-c", composefile, stackname)

	result.Assert(t, icmd.Success)
	stdout := strings.Split(result.Stdout(), "\n")
	expected := strings.Split(string(golden.Get(t, "stack-deploy-with-names.golden")), "\n")
	sort.Strings(stdout)
	sort.Strings(expected)
	assert.DeepEqual(t, stdout, expected)
}
