package stack

import (
	"testing"

	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

func TestStackDeployHelp(t *testing.T) {
	result := icmd.RunCommand("docker", "stack", "deploy", "--help")
	result.Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), "stack-deploy-help.golden")
}
