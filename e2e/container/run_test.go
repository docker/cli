package container

import (
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	shlex "github.com/flynn-archive/go-shlex"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
)

const registryPrefix = "registry:5000"

func TestRunAttachedFromRemoteImageAndRemove(t *testing.T) {
	image := createRemoteImage(t)

	result := icmd.RunCmd(shell(t,
		"docker run --rm %s echo this is output", image))

	result.Assert(t, icmd.Success)
	assert.Check(t, is.Equal("this is output\n", result.Stdout()))
	golden.Assert(t, result.Stderr(), "run-attached-from-remote-and-remove.golden")
}

func TestRunWithContentTrust(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image := fixtures.CreateMaskedTrustedRemoteImage(t, registryPrefix, "trust-run", "latest")

	defer func() {
		icmd.RunCommand("docker", "image", "rm", image).Assert(t, icmd.Success)
	}()

	result := icmd.RunCmd(
		icmd.Command("docker", "run", image),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		Err: fmt.Sprintf("Tagging %s@sha", image[:len(image)-7]),
	})
}

// TODO: create this with registry API instead of engine API
func createRemoteImage(t *testing.T) string {
	image := "registry:5000/alpine:test-run-pulls"
	icmd.RunCommand("docker", "pull", fixtures.AlpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", fixtures.AlpineImage, image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "push", image).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "rmi", image).Assert(t, icmd.Success)
	return image
}

// TODO: move to gotestyourself
func shell(t *testing.T, format string, args ...interface{}) icmd.Cmd {
	cmd, err := shlex.Split(fmt.Sprintf(format, args...))
	assert.NilError(t, err)
	return icmd.Cmd{Command: cmd}
}
