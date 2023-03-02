package context

import (
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/store"
	"gotest.tools/v3/assert"
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
