package orchestrator

import (
	"fmt"
	"os"
	"testing"

	shlex "github.com/flynn-archive/go-shlex"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionWithDefaultOrchestrator(t *testing.T) {
	// Orchestrator by default
	result := icmd.RunCmd(shell(t, "docker version"))
	result.Assert(t, icmd.Success)
	assert.Contains(t, result.Stdout(), "Orchestrator: swarm")
}

func TestVersionWithOverridenEnvOrchestrator(t *testing.T) {
	// Override orchestrator using environment variable
	result := icmd.RunCmd(shell(t, "docker version"), func(cmd *icmd.Cmd) {
		cmd.Env = append(cmd.Env, append(os.Environ(), "DOCKER_ORCHESTRATOR=kubernetes")...)
	})
	result.Assert(t, icmd.Success)
	assert.Contains(t, result.Stdout(), "Orchestrator: kubernetes")
}

func TestVersionWithOverridenConfigOrchestrator(t *testing.T) {
	// Override orchestrator using configuration file
	configDir := fs.NewDir(t, "config", fs.WithFile("config.json", `{"orchestrator": "kubernetes"}`))
	defer configDir.Remove()
	result := icmd.RunCmd(shell(t, fmt.Sprintf("docker --config %s version", configDir.Path())))
	result.Assert(t, icmd.Success)
	assert.Contains(t, result.Stdout(), "Orchestrator: kubernetes")
}

// TODO: move to gotestyourself
func shell(t *testing.T, format string, args ...interface{}) icmd.Cmd {
	cmd, err := shlex.Split(fmt.Sprintf(format, args...))
	require.NoError(t, err)
	return icmd.Cmd{Command: cmd}
}
