package system

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/icmd"
)

// TestInspectInvalidReference migrated from moby/integration-cli
func TestInspectInvalidReference(t *testing.T) {
	// This test should work on both Windows and Linux
	result := icmd.RunCmd(icmd.Command("docker", "inspect", "FooBar"))
	result.Assert(t, icmd.Expected{
		Out:      "[]",
		Err:      "Error: No such object: FooBar",
		ExitCode: 1,
	})
}
