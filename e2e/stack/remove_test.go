package stack

import (
	"strings"
	"testing"

	"github.com/docker/cli/internal/test/environment"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/poll"
)

var pollSettings = environment.DefaultPollSettings

func TestRemove(t *testing.T) {
	stackname := "test-stack-remove"
	deployFullStack(t, stackname)
	defer cleanupFullStack(t, stackname)
	result := icmd.RunCommand("docker", "stack", "rm", stackname)
	result.Assert(t, icmd.Expected{Err: icmd.None})
	golden.Assert(t, result.Stdout(), "stack-remove-success.golden")
}

func deployFullStack(t *testing.T, stackname string) {
	// TODO: this stack should have full options not minimal options
	result := icmd.RunCommand("docker", "stack", "deploy",
		"--compose-file=./testdata/full-stack.yml", stackname)
	result.Assert(t, icmd.Success)

	poll.WaitOn(t, taskCount(stackname, 2), pollSettings)
}

func cleanupFullStack(t *testing.T, stackname string) {
	// FIXME(vdemeester) we shouldn't have to do that. it is hiding a race on docker stack rm
	poll.WaitOn(t, stackRm(stackname), pollSettings)
	poll.WaitOn(t, taskCount(stackname, 0), pollSettings)
}

func stackRm(stackname string) func(t poll.LogT) poll.Result {
	return func(poll.LogT) poll.Result {
		result := icmd.RunCommand("docker", "stack", "rm", stackname)
		if result.Error != nil {
			if strings.Contains(result.Stderr(), "not found") {
				return poll.Success()
			}
			return poll.Continue("docker stack rm %s failed : %v", stackname, result.Error)
		}
		return poll.Success()
	}
}

func taskCount(stackname string, expected int) func(t poll.LogT) poll.Result {
	return func(poll.LogT) poll.Result {
		result := icmd.RunCommand("docker", "stack", "ps", stackname, "-f=desired-state=running")
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
