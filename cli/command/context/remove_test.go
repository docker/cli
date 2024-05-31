package context

import (
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/errdefs"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRemove(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContexts(t, cli, "current", "other")
	assert.NilError(t, RunRemove(cli, RemoveOptions{}, []string{"other"}))
	_, err := cli.ContextStore().GetMetadata("current")
	assert.NilError(t, err)
	_, err = cli.ContextStore().GetMetadata("other")
	assert.Check(t, is.ErrorType(err, errdefs.IsNotFound))
}

func TestRemoveNotAContext(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContexts(t, cli, "current", "other")
	err := RunRemove(cli, RemoveOptions{}, []string{"not-a-context"})
	assert.ErrorContains(t, err, `context "not-a-context" does not exist`)

	err = RunRemove(cli, RemoveOptions{Force: true}, []string{"not-a-context"})
	assert.NilError(t, err)
}

func TestRemoveCurrent(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContexts(t, cli, "current", "other")
	cli.SetCurrentContext("current")
	err := RunRemove(cli, RemoveOptions{}, []string{"current"})
	assert.ErrorContains(t, err, `context "current" is in use, set -f flag to force remove`)
}

func TestRemoveCurrentForce(t *testing.T) {
	configDir := t.TempDir()
	configFilePath := filepath.Join(configDir, "config.json")
	testCfg := configfile.New(configFilePath)
	testCfg.CurrentContext = "current"
	assert.NilError(t, testCfg.Save())

	cli := makeFakeCli(t, withCliConfig(testCfg))
	createTestContexts(t, cli, "current", "other")
	cli.SetCurrentContext("current")
	assert.NilError(t, RunRemove(cli, RemoveOptions{Force: true}, []string{"current"}))
	reloadedConfig, err := config.Load(configDir)
	assert.NilError(t, err)
	assert.Equal(t, "", reloadedConfig.CurrentContext)
}

func TestRemoveDefault(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "other", nil)
	cli.SetCurrentContext("current")
	err := RunRemove(cli, RemoveOptions{}, []string{"default"})
	assert.ErrorContains(t, err, `default: context "default" cannot be removed`)
}
