package fixtures

import (
	"os"
	"testing"

	"github.com/docker/cli/cli/config"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

const (
	// AlpineImage is an image in the test registry
	AlpineImage = "registry:5000/alpine:frozen"
	// BusyboxImage is an image in the test registry
	BusyboxImage = "registry:5000/busybox:frozen"
)

// SetupConfigFile creates a config.json file for testing
func SetupConfigFile(t *testing.T) fs.Dir {
	t.Helper()
	dir := fs.NewDir(t, "trust_test", fs.WithMode(0o700), fs.WithFile("config.json", `{
	"auths": {
		"registry:5000": {
			"auth": "ZWlhaXM6cGFzc3dvcmQK"
		}
	}}`), fs.WithDir("trust", fs.WithDir("private")))
	return *dir
}

// WithConfig sets an environment variable for the docker config location
func WithConfig(dir string) func(cmd *icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		addEnvs(cmd, config.EnvOverrideConfigDir+"="+dir)
	}
}

// WithHome sets the HOME environment variable
func WithHome(path string) func(*icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		addEnvs(cmd, "HOME="+path)
	}
}

// addEnvs adds environment variables to cmd, making sure to preserve the
// current os.Environ(), which would otherwise be omitted (for non-empty .Env).
func addEnvs(cmd *icmd.Cmd, envs ...string) {
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, envs...)
}
