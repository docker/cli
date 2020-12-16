package stack

import (
	"testing"

	"github.com/docker/cli/internal/test/environment"
	"gotest.tools/v3/icmd"
)

func TestConfigFullStack(t *testing.T) {
	environment.SkipIfNotExperimentalDaemon(t)
	result := icmd.RunCommand("docker", "stack", "config", "--compose-file=./testdata/full-stack.yml")
	result.Assert(t, icmd.Success)
}
