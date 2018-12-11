package manager

import (
	"strings"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"gotest.tools/assert"
)

func TestErrPluginNotFound(t *testing.T) {
	var err error = errPluginNotFound("test")
	err.(errPluginNotFound).NotFound()
	assert.Error(t, err, "Error: No such CLI plugin: test")
	assert.Assert(t, IsNotFound(err))
	assert.Assert(t, !IsNotFound(nil))
}

func TestGetPluginDirs(t *testing.T) {
	cli := test.NewFakeCli(nil)

	expected := []string{defaultUserPluginDir}
	expected = append(expected, defaultSystemPluginDirs...)

	assert.Equal(t, strings.Join(expected, ":"), strings.Join(getPluginDirs(cli), ":"))

	extras := []string{
		"foo", "bar", "baz",
	}
	expected = append(extras, expected...)
	cli.SetConfigFile(&configfile.ConfigFile{
		CLIPluginsExtraDirs: extras,
	})
	assert.DeepEqual(t, expected, getPluginDirs(cli))
}
