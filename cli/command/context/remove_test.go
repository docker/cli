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

func TestRemove(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKubeAndSwarm(t, cli, "current", "all")
	createTestContextWithKubeAndSwarm(t, cli, "other", "all")
	assert.NilError(t, RunRemove(cli, RemoveOptions{}, []string{"other"}))
	_, err := cli.ContextStore().GetContextMetadata("current")
	assert.NilError(t, err)
	_, err = cli.ContextStore().GetContextMetadata("other")
	assert.Check(t, store.IsErrContextDoesNotExist(err))
}

func TestRemoveNotAContext(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKubeAndSwarm(t, cli, "current", "all")
	createTestContextWithKubeAndSwarm(t, cli, "other", "all")
	err := RunRemove(cli, RemoveOptions{}, []string{"not-a-context"})
	assert.ErrorContains(t, err, `context "not-a-context" does not exist`)
}

func TestRemoveCurrent(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKubeAndSwarm(t, cli, "current", "all")
	createTestContextWithKubeAndSwarm(t, cli, "other", "all")
	cli.SetCurrentContext("current")
	err := RunRemove(cli, RemoveOptions{}, []string{"current"})
	assert.ErrorContains(t, err, "current: context is in use, set -f flag to force remove")
}

func TestRemoveCurrentForce(t *testing.T) {
	configDir, err := ioutil.TempDir("", t.Name()+"config")
	assert.NilError(t, err)
	defer os.RemoveAll(configDir)
	configFilePath := filepath.Join(configDir, "config.json")
	testCfg := configfile.New(configFilePath)
	testCfg.CurrentContext = "current"
	assert.NilError(t, testCfg.Save())

	cli, cleanup := makeFakeCli(t, withCliConfig(testCfg))
	defer cleanup()
	createTestContextWithKubeAndSwarm(t, cli, "current", "all")
	createTestContextWithKubeAndSwarm(t, cli, "other", "all")
	cli.SetCurrentContext("current")
	assert.NilError(t, RunRemove(cli, RemoveOptions{Force: true}, []string{"current"}))
	reloadedConfig, err := config.Load(configDir)
	assert.NilError(t, err)
	assert.Equal(t, "", reloadedConfig.CurrentContext)
}
