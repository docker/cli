package context

import (
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/store"
	"gotest.tools/v3/assert"
)

func TestRemove(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "current")
	createTestContext(t, cli, "other")
	assert.NilError(t, RunRemove(cli, RemoveOptions{}, []string{"other"}))
	_, err := cli.ContextStore().GetMetadata("current")
	assert.NilError(t, err)
	_, err = cli.ContextStore().GetMetadata("other")
	assert.Check(t, store.IsErrContextDoesNotExist(err))
}

func TestRemoveNotAContext(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "current")
	createTestContext(t, cli, "other")
	err := RunRemove(cli, RemoveOptions{}, []string{"not-a-context"})
	assert.ErrorContains(t, err, `context "not-a-context" does not exist`)
}

func TestRemoveCurrent(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "current")
	createTestContext(t, cli, "other")
	cli.SetCurrentContext("current")
	err := RunRemove(cli, RemoveOptions{}, []string{"current"})
	assert.ErrorContains(t, err, "current: context is in use, set -f flag to force remove")
}

func TestRemoveCurrentForce(t *testing.T) {
	configDir := t.TempDir()
	configFilePath := filepath.Join(configDir, "config.json")
	testCfg := configfile.New(configFilePath)
	testCfg.CurrentContext = "current"
	assert.NilError(t, testCfg.Save())

	cli := makeFakeCli(t, withCliConfig(testCfg))
	createTestContext(t, cli, "current")
	createTestContext(t, cli, "other")
	cli.SetCurrentContext("current")
	assert.NilError(t, RunRemove(cli, RemoveOptions{Force: true}, []string{"current"}))
	reloadedConfig, err := config.Load(configDir)
	assert.NilError(t, err)
	assert.Equal(t, "", reloadedConfig.CurrentContext)
}

func TestRemoveDefault(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "other")
	cli.SetCurrentContext("current")
	err := RunRemove(cli, RemoveOptions{}, []string{"default"})
	assert.ErrorContains(t, err, `default: context "default" cannot be removed`)
}
