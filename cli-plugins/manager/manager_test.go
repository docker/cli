package manager

import (
	"strings"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestListPluginCandidates(t *testing.T) {
	// Populate a selection of directories with various shadowed and bogus/obscure plugin candidates.
	// For the purposes of this test no contents is required and permissions are irrelevant.
	dir := fs.NewDir(t, t.Name(),
		fs.WithDir(
			"plugins1",
			fs.WithFile("docker-plugin1", ""),                        // This appears in each directory
			fs.WithFile("not-a-plugin", ""),                          // Should be ignored
			fs.WithFile("docker-symlinked1", ""),                     // This and ...
			fs.WithSymlink("docker-symlinked2", "docker-symlinked1"), // ... this should both appear
			fs.WithDir("ignored1"),                                   // A directory should be ignored
		),
		fs.WithDir(
			"plugins2",
			fs.WithFile("docker-plugin1", ""),
			fs.WithFile("also-not-a-plugin", ""),
			fs.WithFile("docker-hardlink1", ""),                     // This and ...
			fs.WithHardlink("docker-hardlink2", "docker-hardlink1"), // ... this should both appear
			fs.WithDir("ignored2"),
		),
		fs.WithDir(
			"plugins3-target", // Will be referenced as a symlink from below
			fs.WithFile("docker-plugin1", ""),
			fs.WithDir("ignored3"),
			fs.WithSymlink("docker-brokensymlink", "broken"),           // A broken symlink is still a candidate (but would fail tests later)
			fs.WithFile("non-plugin-symlinked", ""),                    // This shouldn't appear, but ...
			fs.WithSymlink("docker-symlinked", "non-plugin-symlinked"), // ... this link to it should.
		),
		fs.WithSymlink("plugins3", "plugins3-target"),
		fs.WithFile("/plugins4", ""),
		fs.WithSymlink("plugins5", "plugins5-nonexistent-target"),
	)
	defer dir.Remove()

	dirs := make([]string, 0, 6)
	for _, d := range []string{"plugins1", "nonexistent", "plugins2", "plugins3", "plugins4", "plugins5"} {
		dirs = append(dirs, dir.Join(d))
	}

	candidates := listPluginCandidates(dirs)
	exp := map[string][]string{
		"plugin1": {
			dir.Join("plugins1", "docker-plugin1"),
			dir.Join("plugins2", "docker-plugin1"),
			dir.Join("plugins3", "docker-plugin1"),
		},
		"symlinked1": {
			dir.Join("plugins1", "docker-symlinked1"),
		},
		"symlinked2": {
			dir.Join("plugins1", "docker-symlinked2"),
		},
		"hardlink1": {
			dir.Join("plugins2", "docker-hardlink1"),
		},
		"hardlink2": {
			dir.Join("plugins2", "docker-hardlink2"),
		},
		"brokensymlink": {
			dir.Join("plugins3", "docker-brokensymlink"),
		},
		"symlinked": {
			dir.Join("plugins3", "docker-symlinked"),
		},
	}

	assert.DeepEqual(t, candidates, exp)
}

// Regression test for https://github.com/docker/cli/issues/5643.
// Check that inaccessible directories that come before accessible ones are ignored
// and do not prevent the latter from being processed.
func TestListPluginCandidatesInaccesibleDir(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithDir("no-perm", fs.WithMode(0)),
		fs.WithDir("plugins",
			fs.WithFile("docker-buildx", ""),
		),
	)
	defer dir.Remove()

	candidates := listPluginCandidates([]string{
		dir.Join("no-perm"),
		dir.Join("plugins"),
	})
	assert.DeepEqual(t, candidates, map[string][]string{
		"buildx": {
			dir.Join("plugins", "docker-buildx"),
		},
	})
}

func TestGetPlugin(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("docker-bbb", `
#!/bin/sh
echo '{"SchemaVersion":"0.1.0"}'`, fs.WithMode(0o777)),
		fs.WithFile("docker-aaa", `
#!/bin/sh
echo '{"SchemaVersion":"0.1.0"}'`, fs.WithMode(0o777)),
	)
	defer dir.Remove()

	cli := test.NewFakeCli(nil)
	cli.SetConfigFile(&configfile.ConfigFile{CLIPluginsExtraDirs: []string{dir.Path()}})

	plugin, err := GetPlugin("bbb", cli, &cobra.Command{})
	assert.NilError(t, err)
	assert.Equal(t, plugin.Name, "bbb")

	_, err = GetPlugin("ccc", cli, &cobra.Command{})
	assert.Error(t, err, "Error: No such CLI plugin: ccc")
	assert.Assert(t, IsNotFound(err))
}

func TestListPluginsIsSorted(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("docker-bbb", `
#!/bin/sh
echo '{"SchemaVersion":"0.1.0"}'`, fs.WithMode(0o777)),
		fs.WithFile("docker-aaa", `
#!/bin/sh
echo '{"SchemaVersion":"0.1.0"}'`, fs.WithMode(0o777)),
	)
	defer dir.Remove()

	cli := test.NewFakeCli(nil)
	cli.SetConfigFile(&configfile.ConfigFile{CLIPluginsExtraDirs: []string{dir.Path()}})

	plugins, err := ListPlugins(cli, &cobra.Command{})
	assert.NilError(t, err)

	// We're only interested in the plugins we created for testing this, and only
	// if they appear in the expected order
	var names []string
	for _, p := range plugins {
		if p.Name == "aaa" || p.Name == "bbb" {
			names = append(names, p.Name)
		}
	}
	assert.DeepEqual(t, names, []string{"aaa", "bbb"})
}

func TestErrPluginNotFound(t *testing.T) {
	var err error = errPluginNotFound("test")
	err.(errPluginNotFound).NotFound()
	assert.Error(t, err, "Error: No such CLI plugin: test")
	assert.Assert(t, IsNotFound(err))
	assert.Assert(t, !IsNotFound(nil))
}

func TestGetPluginDirs(t *testing.T) {
	cli := test.NewFakeCli(nil)

	pluginDir, err := config.Path("cli-plugins")
	assert.NilError(t, err)
	expected := append([]string{pluginDir}, defaultSystemPluginDirs...)

	var pluginDirs []string
	pluginDirs, err = getPluginDirs(cli.ConfigFile())
	assert.Equal(t, strings.Join(expected, ":"), strings.Join(pluginDirs, ":"))
	assert.NilError(t, err)

	extras := []string{
		"foo", "bar", "baz",
	}
	expected = append(extras, expected...)
	cli.SetConfigFile(&configfile.ConfigFile{
		CLIPluginsExtraDirs: extras,
	})
	pluginDirs, err = getPluginDirs(cli.ConfigFile())
	assert.DeepEqual(t, expected, pluginDirs)
	assert.NilError(t, err)
}
