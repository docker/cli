package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/pkg/errors"
)

const registryPrefix = "registry:5000"

func TestInstallWithContentTrust(t *testing.T) {
	pluginName := fmt.Sprintf("%s/plugin-content-trust", registryPrefix)

	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()

	pluginDir := preparePluginDir(t)
	defer pluginDir.Remove()

	icmd.RunCommand("docker", "plugin", "create", pluginName, pluginDir.Path()).Assert(t, icmd.Success)
	result := icmd.RunCmd(icmd.Command("docker", "plugin", "push", pluginName),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
		fixtures.WithPassphrase("foo", "bar"),
	)
	result.Assert(t, icmd.Expected{
		Out: "Signing and pushing trust metadata",
	})

	icmd.RunCommand("docker", "plugin", "rm", "-f", pluginName).Assert(t, icmd.Success)

	result = icmd.RunCmd(icmd.Command("docker", "plugin", "install", "--grant-all-permissions", pluginName),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		Out: fmt.Sprintf("Status: Downloaded newer image for %s@sha", pluginName),
	})
}

func TestInstallWithContentTrustUntrusted(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()

	result := icmd.RunCmd(icmd.Command("docker", "plugin", "install", "--grant-all-permissions", "tiborvass/sample-volume-plugin:latest"),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "Error: remote trust data does not exist",
	})
}

func preparePluginDir(t *testing.T) *fs.Dir {
	p := &types.PluginConfig{
		Interface: types.PluginConfigInterface{
			Socket: "basic.sock",
			Types:  []types.PluginInterfaceType{{Capability: "docker.dummy/1.0"}},
		},
		Entrypoint: []string{"/basic"},
	}
	configJSON, err := json.Marshal(p)
	assert.NilError(t, err)

	binPath, err := ensureBasicPluginBin()
	assert.NilError(t, err)

	dir := fs.NewDir(t, "plugin_test",
		fs.WithFile("config.json", string(configJSON), fs.WithMode(0644)),
		fs.WithDir("rootfs", fs.WithMode(0755)),
	)
	icmd.RunCommand("/bin/cp", binPath, dir.Join("rootfs", p.Entrypoint[0])).Assert(t, icmd.Success)
	return dir
}

func ensureBasicPluginBin() (string, error) {
	name := "docker-basic-plugin"
	p, err := exec.LookPath(name)
	if err == nil {
		return p, nil
	}

	goBin, err := exec.LookPath("/usr/local/go/bin/go")
	if err != nil {
		return "", err
	}
	installPath := filepath.Join(os.Getenv("GOPATH"), "bin", name)
	cmd := exec.Command(goBin, "build", "-o", installPath, "./basic")
	cmd.Env = append(cmd.Env, "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", errors.Wrapf(err, "error building basic plugin bin: %s", string(out))
	}
	return installPath, nil
}
