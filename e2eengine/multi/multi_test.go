package multi

import (
	"os"
	"testing"

	"github.com/docker/cli/e2eengine"

	"gotest.tools/icmd"
)

func TestDockerEngineOnContainerdMultiTest(t *testing.T) {
	defer func() {
		err := e2eengine.CleanupEngine(t)
		if err != nil {
			t.Errorf("Failed to cleanup engine: %s", err)
		}
	}()

	t.Log("Attempt engine init without experimental")
	// First init
	result := icmd.RunCmd(icmd.Command("docker", "engine", "init"),
		func(c *icmd.Cmd) {
			c.Env = append(c.Env, "DOCKER_CLI_EXPERIMENTAL=disabled")
		})
	result.Assert(t, icmd.Expected{
		Out:      "",
		Err:      "docker engine init is only supported",
		ExitCode: 1,
	})

	t.Log("First engine init")
	// First init
	result = icmd.RunCmd(icmd.Command("docker", "engine", "init"),
		func(c *icmd.Cmd) {
			c.Env = append(c.Env, "DOCKER_CLI_EXPERIMENTAL=enabled")
		})
	result.Assert(t, icmd.Expected{
		Out:      "Success!  The docker engine is now running.",
		Err:      "",
		ExitCode: 0,
	})

	t.Log("checking for updates")
	// Check for updates
	result = icmd.RunCmd(icmd.Command("docker", "engine", "check", "--downgrades", "--pre-releases"))
	result.Assert(t, icmd.Expected{
		Out:      "VERSION",
		Err:      "",
		ExitCode: 0,
	})

	t.Log("attempt second init (should fail)")
	// Attempt to init a second time and fail
	result = icmd.RunCmd(icmd.Command("docker", "engine", "init"),
		func(c *icmd.Cmd) {
			c.Env = append(c.Env, "DOCKER_CLI_EXPERIMENTAL=enabled")
		})
	result.Assert(t, icmd.Expected{
		Out:      "",
		Err:      "engine already present",
		ExitCode: 1,
	})

	t.Log("perform update")
	// Now update and succeed
	targetVersion := os.Getenv("VERSION")
	result = icmd.RunCmd(icmd.Command("docker", "engine", "update", "--version", targetVersion))
	result.Assert(t, icmd.Expected{
		Out:      "Success!  The docker engine is now running.",
		Err:      "",
		ExitCode: 0,
	})

	t.Log("remove engine")
	result = icmd.RunCmd(icmd.Command("docker", "engine", "rm"),
		func(c *icmd.Cmd) {
			c.Env = append(c.Env, "DOCKER_CLI_EXPERIMENTAL=enabled")
		})
	result.Assert(t, icmd.Expected{
		Out:      "",
		Err:      "",
		ExitCode: 0,
	})
}
