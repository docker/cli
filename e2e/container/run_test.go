package container

import (
	"fmt"
	"testing"

	shlex "github.com/flynn-archive/go-shlex"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const alpineImage = "registry:5000/alpine:3.6"

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
	icmd.RunCommand("docker", "pull", alpineImage).Assert(t, icmd.Success)
	icmd.RunCommand("docker", "tag", alpineImage, image).Assert(t, icmd.Success)
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
