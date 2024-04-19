package testutils

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

//go:embed plugins/*
var plugins embed.FS

// SetupPlugin builds a plugin and creates a temporary
// directory with the plugin's config.json and rootfs.
func SetupPlugin(t *testing.T, ctx context.Context) *fs.Dir {
	t.Helper()

	p := &types.PluginConfig{
		Linux: types.PluginConfigLinux{
			Capabilities: []string{"CAP_SYS_ADMIN"},
		},
		Interface: types.PluginConfigInterface{
			Socket: "basic.sock",
			Types:  []types.PluginInterfaceType{{Capability: "docker.dummy/1.0"}},
		},
		Entrypoint: []string{"/basic"},
	}
	configJSON, err := json.Marshal(p)
	assert.NilError(t, err)

	binPath, err := buildPlugin(t, ctx)
	assert.NilError(t, err)

	dir := fs.NewDir(t, "plugin_test",
		fs.WithFile("config.json", string(configJSON), fs.WithMode(0o644)),
		fs.WithDir("rootfs", fs.WithMode(0o755)),
	)

	icmd.RunCommand("/bin/cp", binPath, dir.Join("rootfs", p.Entrypoint[0])).Assert(t, icmd.Success)
	return dir
}

// buildPlugin uses Go to build a plugin from one of the source files in the plugins directory.
// It returns the path to the built plugin binary.
// To allow for multiple plugins to be built in parallel, the plugin is compiled with a unique
// identifier in the binary. This is done by setting a linker flag with the -ldflags option.
func buildPlugin(t *testing.T, ctx context.Context) (string, error) {
	t.Helper()

	randomName, err := randomString()
	if err != nil {
		return "", err
	}

	goBin, err := exec.LookPath("/usr/local/go/bin/go")
	if err != nil {
		return "", err
	}
	installPath := filepath.Join(os.Getenv("GOPATH"), "bin", randomName)

	pluginContent, err := plugins.ReadFile("plugins/basic.go")
	if err != nil {
		return "", err
	}
	dir := fs.NewDir(t, "plugin_build")
	if err := os.WriteFile(dir.Join("main.go"), pluginContent, 0o644); err != nil {
		return "", err
	}
	defer dir.Remove()

	cmd := exec.CommandContext(ctx, goBin, "build", "-ldflags",
		fmt.Sprintf("-X 'main.UNIQUEME=%s'", randomName),
		"-o", installPath, dir.Join("main.go"))

	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	if out, err := cmd.CombinedOutput(); err != nil {
		return "", errors.Wrapf(err, "error building basic plugin bin: %s", string(out))
	}

	return installPath, nil
}

func randomString() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
