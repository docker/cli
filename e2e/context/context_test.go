package context

import (
	"testing"

	"gotest.tools/golden"
	"gotest.tools/icmd"
)

func TestContextList(t *testing.T) {
	cmd := icmd.Command("docker", "context", "ls")
	cmd.Env = append(cmd.Env,
		"DOCKER_CONFIG=./testdata/test-dockerconfig",
		"KUBECONFIG=./testdata/test-kubeconfig",
	)
	result := icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		Err:      icmd.None,
		ExitCode: 0,
	})
	golden.Assert(t, result.Stdout(), "context-ls.golden")
}
