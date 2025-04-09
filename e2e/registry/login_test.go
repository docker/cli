package registry

import (
	"io"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"gotest.tools/v3/assert"
)

func TestOauthLogin(t *testing.T) {
	t.Parallel()
	loginCmd := exec.Command("docker", "login")

	p, err := pty.Start(loginCmd)
	assert.NilError(t, err)
	defer func() {
		_ = loginCmd.Wait()
		_ = p.Close()
	}()

	time.Sleep(1 * time.Second)
	pid := loginCmd.Process.Pid
	t.Logf("terminating PID %d", pid)
	err = syscall.Kill(pid, syscall.SIGTERM)
	assert.NilError(t, err)

	output, _ := io.ReadAll(p)
	assert.Check(t, strings.Contains(string(output), "USING WEB-BASED LOGIN"), string(output))
}

func TestLoginWithEscapeHatch(t *testing.T) {
	t.Parallel()
	loginCmd := exec.Command("docker", "login")
	loginCmd.Env = append(loginCmd.Env, "DOCKER_CLI_DISABLE_OAUTH_LOGIN=1")

	p, err := pty.Start(loginCmd)
	assert.NilError(t, err)
	defer func() {
		_ = loginCmd.Wait()
		_ = p.Close()
	}()

	time.Sleep(1 * time.Second)
	pid := loginCmd.Process.Pid
	t.Logf("terminating PID %d", pid)
	err = syscall.Kill(pid, syscall.SIGTERM)
	assert.NilError(t, err)

	output, _ := io.ReadAll(p)
	assert.Check(t, strings.Contains(string(output), "Username:"), string(output))
}
