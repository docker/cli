package stack

import (
	"fmt"
	"strings"
	"testing"
	"time"

	shlex "github.com/flynn-archive/go-shlex"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/stretchr/testify/require"
)

func TestRemove(t *testing.T) {
	stackname := "test-stack-remove"
	deployFullStack(t, stackname)
	defer cleanupFullStack(t, stackname)

	result := icmd.RunCmd(shell(t, "docker stack rm %s", stackname))

	result.Assert(t, icmd.Expected{Err: icmd.None})
	golden.Assert(t, result.Stdout(), "stack-remove-success.golden")
}

func deployFullStack(t *testing.T, stackname string) {
	// TODO: this stack should have full options not minimal options
	result := icmd.RunCmd(shell(t,
		"docker stack deploy --compose-file=./testdata/full-stack.yml %s", stackname))
	result.Assert(t, icmd.Success)

	waitOn(t, taskCount(stackname, 2), 0)
}

func cleanupFullStack(t *testing.T, stackname string) {
	result := icmd.RunCmd(shell(t, "docker stack rm %s", stackname))
	result.Assert(t, icmd.Success)
	waitOn(t, taskCount(stackname, 0), 0)
}

func taskCount(stackname string, expected int) func() (bool, error) {
	return func() (bool, error) {
		result := icmd.RunCommand(
			"docker", "stack", "ps", "-f=desired-state=running", stackname)
		count := lines(result.Stdout()) - 1
		return count == expected, nil
	}
}

func lines(out string) int {
	return len(strings.Split(strings.TrimSpace(out), "\n"))
}

// TODO: move to gotestyourself
func shell(t *testing.T, format string, args ...interface{}) icmd.Cmd {
	cmd, err := shlex.Split(fmt.Sprintf(format, args...))
	require.NoError(t, err)
	return icmd.Cmd{Command: cmd}
}

// TODO: move to gotestyourself
func waitOn(t *testing.T, check func() (bool, error), timeout time.Duration) {
	if timeout == time.Duration(0) {
		timeout = defaultTimeout()
	}

	after := time.After(timeout)
	for {
		select {
		case <-after:
			// TODO: include check function name in error message
			t.Fatalf("timeout hit after %s", timeout)
		default:
			// TODO: maybe return a failure message as well?
			done, err := check()
			if done {
				return
			}
			if err != nil {
				t.Fatal(err.Error())
			}
		}
	}
}

func defaultTimeout() time.Duration {
	// TODO: support override from environment variable
	return 10 * time.Second
}
