package stack

import (
	"testing"

	"github.com/docker/cli/e2e/internal/stack"
	"github.com/docker/cli/internal/test/environment"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
)

var pollSettings = environment.DefaultPollSettings

func TestRemove(t *testing.T) {
	stackname := "test-stack-remove"
	stack.DeployFullStack(t, pollSettings, stackname)
	defer stack.CleanupFullStack(t, pollSettings, stackname)

	result := icmd.RunCmd(environment.Shell(t, "docker stack rm %s", stackname))

	result.Assert(t, icmd.Expected{Err: icmd.None})
	golden.Assert(t, result.Stdout(), "stack-remove-success.golden")
}
