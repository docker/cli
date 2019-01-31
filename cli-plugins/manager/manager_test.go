package manager

import (
	"strings"
	"testing"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"gotest.tools/assert"
	"gotest.tools/fs"
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

	var dirs []string
	for _, d := range []string{"plugins1", "nonexistent", "plugins2", "plugins3", "plugins4", "plugins5"} {
		dirs = append(dirs, dir.Join(d))
	}

	candidates, err := listPluginCandidates(dirs)
	assert.NilError(t, err)
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

func TestErrPluginNotFound(t *testing.T) {
	var err error = errPluginNotFound("test")
	err.(errPluginNotFound).NotFound()
	assert.Error(t, err, "Error: No such CLI plugin: test")
	assert.Assert(t, IsNotFound(err))
	assert.Assert(t, !IsNotFound(nil))
}

func TestGetPluginDirs(t *testing.T) {
	cli := test.NewFakeCli(nil)

	expected := []string{config.Path("cli-plugins")}
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
