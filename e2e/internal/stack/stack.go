package stack

import (
	"strings"
	"testing"

	"github.com/docker/cli/internal/test/environment"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/gotestyourself/gotestyourself/poll"
)

// DeployFullStack run docker stack deploy with specified options
func DeployFullStack(t *testing.T, pollSettings poll.SettingOp, stackname string, ops ...icmd.CmdOp) {
	// TODO: this stack should have full options not minimal options
	result := icmd.RunCmd(
		environment.Shell(t, "docker stack deploy --compose-file=./testdata/full-stack.yml %s", stackname),
		ops...,
	)
	result.Assert(t, icmd.Success)
	poll.WaitOn(t, taskCount(t, stackname, 2, ops...), pollSettings)
}

// CleanupFullStack cleans the stack with specified options
func CleanupFullStack(t *testing.T, pollSettings poll.SettingOp, stackname string, ops ...icmd.CmdOp) {
	// FIXME(vdemeester) we shouldn't have to do that. it is hidding a race on docker stack rm
	poll.WaitOn(t, stackRm(t, stackname, ops...), pollSettings)
	poll.WaitOn(t, taskCount(t, stackname, 0, ops...), pollSettings)
}

func stackRm(t *testing.T, stackname string, ops ...icmd.CmdOp) func(t poll.LogT) poll.Result {
	return func(l poll.LogT) poll.Result {
		result := icmd.RunCmd(
			environment.Shell(t, "docker stack rm %s", stackname),
			ops...,
		)
		if result.Error != nil {
			return poll.Continue("docker stack rm %s failed : %v", stackname, result.Error)
		}
		return poll.Success()
	}
}

func taskCount(t *testing.T, stackname string, expected int, ops ...icmd.CmdOp) func(t poll.LogT) poll.Result {
	return func(l poll.LogT) poll.Result {
		result := icmd.RunCmd(
			environment.Shell(t, "docker stack ps %s", stackname),
			ops...,
		)
		count := lines(result.Stdout()) - 1
		if count == expected {
			return poll.Success()
		}
		return poll.Continue("task count is %d waiting for %d", count, expected)
	}
}

func lines(out string) int {
	return len(strings.Split(strings.TrimSpace(out), "\n"))
}
