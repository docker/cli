package context

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/store"
	"gotest.tools/assert"
)

func TestUse(t *testing.T) {
	configDir, err := ioutil.TempDir("", t.Name()+"config")
	assert.NilError(t, err)
	defer os.RemoveAll(configDir)
	configFilePath := filepath.Join(configDir, "config.json")
	testCfg := configfile.New(configFilePath)
	cli, cleanup := makeFakeCli(t, withCliConfig(testCfg))
	defer cleanup()
	err = runCreate(cli, &createOptions{
		name:   "test",
		docker: map[string]string{},
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
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	err := newUseCommand(cli).RunE(nil, []string{"test"})
	assert.Check(t, store.IsErrContextDoesNotExist(err))
}
