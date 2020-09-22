package stack

import (
	"testing"

	"gotest.tools/v3/icmd"
)

func TestConfigFullStack(t *testing.T) {
	result := icmd.RunCommand("docker", "stack", "config", "--compose-file=./testdata/full-stack.yml")
	result.Assert(t, icmd.Success)
}
