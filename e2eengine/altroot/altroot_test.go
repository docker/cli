package check

import (
	"os"
	"testing"

	"github.com/docker/cli/e2eengine"

	"gotest.tools/icmd"
)

func TestDockerEngineOnContainerdAltRootConfig(t *testing.T) {
	defer func() {
		err := e2eengine.CleanupEngine(t)
		if err != nil {
			t.Errorf("Failed to cleanup engine: %s", err)
		}
	}()

	t.Log("First engine init")
	// First init
	result := icmd.RunCmd(icmd.Command("docker", "engine", "init", "--config-file", "/tmp/etc/docker/daemon.json"),
		func(c *icmd.Cmd) {
			c.Env = append(c.Env, "DOCKER_CLI_EXPERIMENTAL=enabled")
		})
	result.Assert(t, icmd.Expected{
		Out:      "Success!  The docker engine is now running.",
		Err:      "",
		ExitCode: 0,
	})

	// Make sure update doesn't blow up with alternate config path
	t.Log("perform update")
	// Now update and succeed
	targetVersion := os.Getenv("VERSION")
	result = icmd.RunCmd(icmd.Command("docker", "engine", "update", "--version", targetVersion))
	result.Assert(t, icmd.Expected{
		Out:      "Success!  The docker engine is now running.",
		Err:      "",
		ExitCode: 0,
	})
}
