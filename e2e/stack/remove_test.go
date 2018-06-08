package stack

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test/environment"
	"gotest.tools/golden"
	"gotest.tools/icmd"
	"gotest.tools/poll"
	"gotest.tools/skip"
)

var pollSettings = environment.DefaultPollSettings

func TestRemove(t *testing.T) {
	t.Run("Swarm", func(t *testing.T) {
		testRemove(t, "swarm")
	})
	t.Run("Kubernetes", func(t *testing.T) {
		skip.If(t, !environment.KubernetesEnabled())

		testRemove(t, "kubernetes")
	})
}

func testRemove(t *testing.T, orchestrator string) {
	stackname := "test-stack-remove-" + orchestrator
	deployFullStack(t, orchestrator, stackname)
	defer cleanupFullStack(t, orchestrator, stackname)
	result := icmd.RunCommand("docker", "--orchestrator", orchestrator,
		"stack", "rm", stackname)
	result.Assert(t, icmd.Expected{Err: icmd.None})
	golden.Assert(t, result.Stdout(),
		fmt.Sprintf("stack-remove-%s-success.golden", orchestrator))
}

func deployFullStack(t *testing.T, orchestrator, stackname string) {
	// TODO: this stack should have full options not minimal options
	result := icmd.RunCommand("docker", "--orchestrator", orchestrator,
		"stack", "deploy", "--compose-file=./testdata/full-stack.yml", stackname)
	result.Assert(t, icmd.Success)

	poll.WaitOn(t, taskCount(orchestrator, stackname, 2), pollSettings)
}

func cleanupFullStack(t *testing.T, orchestrator, stackname string) {
	// FIXME(vdemeester) we shouldn't have to do that. it is hidding a race on docker stack rm
	poll.WaitOn(t, stackRm(orchestrator, stackname), pollSettings)
	poll.WaitOn(t, taskCount(orchestrator, stackname, 0), pollSettings)
}

func stackRm(orchestrator, stackname string) func(t poll.LogT) poll.Result {
	return func(poll.LogT) poll.Result {
		result := icmd.RunCommand("docker", "--orchestrator", orchestrator, "stack", "rm", stackname)
		if result.Error != nil {
			if strings.Contains(result.Stderr(), "not found") {
				return poll.Success()
			}
			return poll.Continue("docker stack rm %s failed : %v", stackname, result.Error)
		}
		return poll.Success()
	}
}

func taskCount(orchestrator, stackname string, expected int) func(t poll.LogT) poll.Result {
	return func(poll.LogT) poll.Result {
		args := []string{"--orchestrator", orchestrator, "stack", "ps", stackname}
		// FIXME(chris-crone): remove when we support filtering by desired-state on kubernetes
		if orchestrator == "swarm" {
			args = append(args, "-f=desired-state=running")
		}
		result := icmd.RunCommand("docker", args...)
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
