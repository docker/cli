package context

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/cli/cli/flags"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestUse(t *testing.T) {
	configDir := t.TempDir()
	configFilePath := filepath.Join(configDir, "config.json")
	testCfg := configfile.New(configFilePath)
	cli := makeFakeCli(t, withCliConfig(testCfg))
	err := RunCreate(cli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
	})
	assert.NilError(t, err)
	assert.NilError(t, newUseCommand(cli).RunE(nil, []string{"test"}))
	reloadedConfig, err := config.Load(configDir)
	assert.NilError(t, err)
	assert.Equal(t, "test", reloadedConfig.CurrentContext)

	// switch back to default
	cli.OutBuffer().Reset()
	cli.ErrBuffer().Reset()
	assert.NilError(t, newUseCommand(cli).RunE(nil, []string{"default"}))
	reloadedConfig, err = config.Load(configDir)
	assert.NilError(t, err)
	assert.Equal(t, "", reloadedConfig.CurrentContext)
	assert.Equal(t, "default\n", cli.OutBuffer().String())
	assert.Equal(t, "Current context is now \"default\"\n", cli.ErrBuffer().String())
}

func TestUseNoExist(t *testing.T) {
	cli := makeFakeCli(t)
	err := newUseCommand(cli).RunE(nil, []string{"test"})
	assert.Check(t, store.IsErrContextDoesNotExist(err))
}

func TestUseHostOverride(t *testing.T) {
	t.Setenv("DOCKER_HOST", "tcp://ed:2375/")

	configDir := t.TempDir()
	configFilePath := filepath.Join(configDir, "config.json")
	testCfg := configfile.New(configFilePath)
	cli := makeFakeCli(t, withCliConfig(testCfg))
	err := RunCreate(cli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{},
	})
	assert.NilError(t, err)

	cli.ResetOutputBuffers()
	err = newUseCommand(cli).RunE(nil, []string{"test"})
	assert.NilError(t, err)
	assert.Assert(t, is.Contains(
		cli.ErrBuffer().String(),
		`Warning: DOCKER_HOST environment variable overrides the active context.`,
	))
	assert.Assert(t, is.Contains(cli.ErrBuffer().String(), `Current context is now "test"`))
	assert.Equal(t, cli.OutBuffer().String(), "test\n")

	// setting DOCKER_HOST with the default context should not print a warning
	cli.ResetOutputBuffers()
	err = newUseCommand(cli).RunE(nil, []string{"default"})
	assert.NilError(t, err)
	assert.Assert(t, is.Contains(cli.ErrBuffer().String(), `Current context is now "default"`))
	assert.Equal(t, cli.OutBuffer().String(), "default\n")
}

// An empty DOCKER_HOST used to break the 'context use' flow.
// So we have a test with fewer fakes that tests this flow holistically.
// https://github.com/docker/cli/issues/3667
func TestUseHostOverrideEmpty(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")

	configDir := t.TempDir()
	config.SetDir(configDir)

	socketPath := "unix://" + filepath.Join(configDir, "docker.sock")

	var out *bytes.Buffer
	var cli *command.DockerCli

	loadCli := func() {
		out = bytes.NewBuffer(nil)

		var err error
		cli, err = command.NewDockerCli(command.WithCombinedStreams(out))
		assert.NilError(t, err)
		assert.NilError(t, cli.Initialize(flags.NewClientOptions()))
	}
	loadCli()
	err := RunCreate(cli, &CreateOptions{
		Name:   "test",
		Docker: map[string]string{"host": socketPath},
	})
	assert.NilError(t, err)

	err = newUseCommand(cli).RunE(nil, []string{"test"})
	assert.NilError(t, err)
	assert.Assert(t, !is.Contains(out.String(), "Warning")().Success())
	assert.Assert(t, is.Contains(out.String(), `Current context is now "test"`))

	loadCli()
	err = newShowCommand(cli).RunE(nil, nil)
	assert.NilError(t, err)
	assert.Assert(t, is.Contains(out.String(), "test"))

	apiclient := cli.Client()
	assert.Equal(t, apiclient.DaemonHost(), socketPath)
}
