package context

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/flags"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestUse(t *testing.T) {
	configDir := t.TempDir()
	configFilePath := filepath.Join(configDir, "config.json")
	testCfg := configfile.New(configFilePath)
	fakeCli := makeFakeCli(t, withCliConfig(testCfg))
	err := RunCreate(fakeCli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
	})
	assert.NilError(t, err)
	assert.NilError(t, newUseCommand(fakeCli).RunE(nil, []string{"test"}))
	reloadedConfig, err := config.Load(configDir)
	assert.NilError(t, err)
	assert.Equal(t, "test", reloadedConfig.CurrentContext)

	// switch back to default
	fakeCli.OutBuffer().Reset()
	fakeCli.ErrBuffer().Reset()
	assert.NilError(t, newUseCommand(fakeCli).RunE(nil, []string{"default"}))
	reloadedConfig, err = config.Load(configDir)
	assert.NilError(t, err)
	assert.Equal(t, "", reloadedConfig.CurrentContext)
	assert.Equal(t, "default\n", fakeCli.OutBuffer().String())
	assert.Equal(t, "Current context is now \"default\"\n", fakeCli.ErrBuffer().String())
}

func TestUseNoExist(t *testing.T) {
	fakeCli := makeFakeCli(t)
	err := newUseCommand(fakeCli).RunE(nil, []string{"test"})
	assert.Check(t, is.ErrorType(err, errdefs.IsNotFound))
}

// TestUseDefaultWithoutConfigFile verifies that the CLI does not create
// the default config file and directory when using the default context.
func TestUseDefaultWithoutConfigFile(t *testing.T) {
	// We must use a temporary home-directory, because this test covers
	// the _default_ configuration file. If we specify a custom configuration
	// file, the CLI produces an error if the file doesn't exist.
	tmpHomeDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpHomeDir)
	} else {
		t.Setenv("HOME", tmpHomeDir)
	}
	configDir := filepath.Join(tmpHomeDir, ".docker")
	configFilePath := filepath.Join(configDir, "config.json")

	// Verify config-dir and -file don't exist before
	_, err := os.Stat(configDir)
	assert.Check(t, errors.Is(err, os.ErrNotExist))
	_, err = os.Stat(configFilePath)
	assert.Check(t, errors.Is(err, os.ErrNotExist))

	fakeCli, err := cli.NewDockerCli(cli.WithCombinedStreams(io.Discard))
	assert.NilError(t, err)
	assert.NilError(t, newUseCommand(fakeCli).RunE(nil, []string{"default"}))

	// Verify config-dir and -file don't exist after
	_, err = os.Stat(configDir)
	assert.Check(t, errors.Is(err, os.ErrNotExist))
	_, err = os.Stat(configFilePath)
	assert.Check(t, errors.Is(err, os.ErrNotExist))
}

func TestUseHostOverride(t *testing.T) {
	t.Setenv("DOCKER_HOST", "tcp://ed:2375/")

	configDir := t.TempDir()
	configFilePath := filepath.Join(configDir, "config.json")
	testCfg := configfile.New(configFilePath)
	fakeCli := makeFakeCli(t, withCliConfig(testCfg))
	err := RunCreate(fakeCli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
	})
	assert.NilError(t, err)

	fakeCli.ResetOutputBuffers()
	err = newUseCommand(fakeCli).RunE(nil, []string{"test"})
	assert.NilError(t, err)
	assert.Assert(t, is.Contains(
		fakeCli.ErrBuffer().String(),
		`Warning: DOCKER_HOST environment variable overrides the active context.`,
	))
	assert.Assert(t, is.Contains(fakeCli.ErrBuffer().String(), `Current context is now "test"`))
	assert.Equal(t, fakeCli.OutBuffer().String(), "test\n")

	// setting DOCKER_HOST with the default context should not print a warning
	fakeCli.ResetOutputBuffers()
	err = newUseCommand(fakeCli).RunE(nil, []string{"default"})
	assert.NilError(t, err)
	assert.Assert(t, is.Contains(fakeCli.ErrBuffer().String(), `Current context is now "default"`))
	assert.Equal(t, fakeCli.OutBuffer().String(), "default\n")
}

// An empty DOCKER_HOST used to break the 'context use' flow.
// So we have a test with fewer fakes that tests this flow holistically.
// https://github.com/docker/cli/issues/3667
func TestUseHostOverrideEmpty(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")

	configDir := t.TempDir()
	config.SetDir(configDir)

	socketPath := "unix://" + filepath.Join(configDir, "docker.sock")

	var out bytes.Buffer

	dockerCLI, err := cli.NewDockerCli(cli.WithCombinedStreams(&out))
	assert.NilError(t, err)
	assert.NilError(t, dockerCLI.Initialize(flags.NewClientOptions()))
	err = RunCreate(dockerCLI, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{"host": socketPath},
	})
	assert.NilError(t, err)

	err = newUseCommand(dockerCLI).RunE(nil, []string{"test"})
	assert.NilError(t, err)
	assert.Assert(t, !is.Contains(out.String(), "Warning")().Success())
	assert.Assert(t, is.Contains(out.String(), `Current context is now "test"`))

	out.Reset()
	dockerCLI, err = cli.NewDockerCli(cli.WithCombinedStreams(&out))
	assert.NilError(t, err)
	assert.NilError(t, dockerCLI.Initialize(flags.NewClientOptions()))

	err = newShowCommand(dockerCLI).RunE(nil, nil)
	assert.NilError(t, err)
	assert.Assert(t, is.Contains(out.String(), "test"))

	apiclient := dockerCLI.Client()
	assert.Equal(t, apiclient.DaemonHost(), socketPath)
}
