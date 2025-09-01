package context

import (
	"path/filepath"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRemove(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContexts(t, cli, "current", "other")
	assert.NilError(t, runRemove(cli, removeOptions{}, []string{"other"}))
	_, err := cli.ContextStore().GetMetadata("current")
	assert.NilError(t, err)
	_, err = cli.ContextStore().GetMetadata("other")
	assert.Check(t, is.ErrorType(err, errdefs.IsNotFound))
}

func TestRemoveNotAContext(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContexts(t, cli, "current", "other")
	err := runRemove(cli, removeOptions{}, []string{"not-a-context"})
	assert.ErrorContains(t, err, `context "not-a-context" does not exist`)

	err = runRemove(cli, removeOptions{force: true}, []string{"not-a-context"})
	assert.NilError(t, err)
}

func TestRemoveCurrent(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContexts(t, cli, "current", "other")
	cli.SetCurrentContext("current")
	err := runRemove(cli, removeOptions{}, []string{"current"})
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
	assert.NilError(t, runRemove(cli, removeOptions{force: true}, []string{"current"}))
	reloadedConfig, err := config.Load(configDir)
	assert.NilError(t, err)
	assert.Equal(t, "", reloadedConfig.CurrentContext)
}

func TestRemoveDefault(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "other", nil)
	cli.SetCurrentContext("current")
	err := runRemove(cli, removeOptions{}, []string{"default"})
	assert.ErrorContains(t, err, `context "default" cannot be removed`)
}
