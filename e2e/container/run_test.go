package container

import (
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	shlex "github.com/flynn-archive/go-shlex"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAttachedFromRemoteImageAndRemove(t *testing.T) {
	image := createRemoteImage(t)

	result := icmd.RunCmd(shell(t,
		"docker run --rm %s echo this is output", image))

	result.Assert(t, icmd.Success)
	assert.Equal(t, "this is output\n", result.Stdout())
	golden.Assert(t, result.Stderr(), "run-attached-from-remote-and-remove.golden")
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
	require.NoError(t, err)
	return icmd.Cmd{Command: cmd}
}
