package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

func TestRemoveOne(t *testing.T) {
	const buildCtx = "one"
	const coolApp = "cool"
	const coolScript = `#!/bin/sh
	echo "Running 'cool' app ..."
	uname -a
	`
	dir := fs.NewDir(t, "test-install-single-file",
		fs.WithDir(buildCtx,
			fs.WithFile("Dockerfile", fmt.Sprintf(`
			FROM %s
			COPY cool /egress/%s
			CMD ["echo", "'cool' app successfully built!"]
			`, fixtures.AlpineImage, coolApp)),
			fs.WithFile("cool", coolScript, fs.WithMode(0o755)),
		),
	)
	defer dir.Remove()

	appBase := fs.NewDir(t, "docker-app-base",
		fs.WithDir("bin",
			fs.WithMode(os.FileMode(0o755))),
	)
	defer appBase.Remove()

	installedLink := filepath.Join(appBase.Path(), "bin", coolApp)
	installedApp := filepath.Join(appBase.Path(), "pkg", "file", dir.Path(), buildCtx)
	t.Setenv("DOCKER_APP_BASE", appBase.Path())

	result := icmd.RunCmd(
		icmd.Command("docker", "app", "install", buildCtx),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)
	assert.Check(t, fileExists(installedLink))
	assert.Check(t, fileExists(installedApp))

	result = icmd.RunCmd(
		icmd.Command("docker", "app", "remove", buildCtx),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)
	assert.Check(t, !fileExists(installedLink))
	assert.Check(t, !fileExists(installedApp))
	assert.Check(t, !fileExists(filepath.Join(appBase.Path(), "pkg", "file")))
	assert.Check(t, fileExists(filepath.Join(appBase.Path(), "pkg")))
}

func TestRemoveMulti(t *testing.T) {
	const buildCtx = "multi"
	const runScript = `#!/bin/sh
	echo "Running 'multi' ..."
	uname -a
	##	
	`
	dir := fs.NewDir(t, "test-install-multi-file",
		fs.WithDir(buildCtx,
			fs.WithFile("Dockerfile", fmt.Sprintf(`
			FROM %s
			COPY . /egress
			CMD ["echo", "'multi' app successfully built!"]
			`, fixtures.AlpineImage)),
			fs.WithFile(".dockerignore", `
			Dockerfile
			.dockerignore
			`, fs.WithMode(0o644)),
			fs.WithFile("LICENSE", "", fs.WithMode(0o644)),
			fs.WithFile("README.md", "", fs.WithMode(0o644)),
			fs.WithFile("run", runScript, fs.WithMode(0o755)),
		),
	)
	defer dir.Remove()

	appBase := fs.NewDir(t, "docker-app-base",
		fs.WithDir("bin",
			fs.WithMode(os.FileMode(0o755))),
		fs.WithDir("pkg",
			fs.WithMode(os.FileMode(0o755))),
	)
	defer appBase.Remove()

	installedLink := filepath.Join(appBase.Path(), "bin", buildCtx)
	installedApp := filepath.Join(appBase.Path(), "pkg", "file", dir.Path(), buildCtx)
	t.Setenv("DOCKER_APP_BASE", appBase.Path())

	result := icmd.RunCmd(
		icmd.Command("docker", "app", "install", buildCtx),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)
	assert.Check(t, fileExists(installedLink))
	assert.Check(t, fileExists(installedApp))

	result = icmd.RunCmd(
		icmd.Command("docker", "app", "remove", buildCtx),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)
	assert.Check(t, !fileExists(installedLink))
	assert.Check(t, !fileExists(installedApp))
	assert.Check(t, !fileExists(filepath.Join(appBase.Path(), "pkg", "file")))
	assert.Check(t, fileExists(filepath.Join(appBase.Path(), "pkg")))
}

func TestRemoveCustom(t *testing.T) {
	const buildCtx = "custom"

	cwd, err := os.Getwd()
	assert.NilError(t, err, "failed to get cwd for test")

	// custom install/uninstall scripts will create/remove this
	customDir := fs.NewDir(t, "custom")
	customFile := filepath.Join(customDir.Path(), "custom.file")

	installScript := fmt.Sprintf(`#!/bin/sh
	set -e
	echo "Installing 'custom' app ..."
	export PATH=/bin:/usr/bin
	cd '%s' && pwd
	CUSTOM_DIR='%s'
	mkdir -p $CUSTOM_DIR
	cd $CUSTOM_DIR && pwd
	touch custom.file
	echo "'Custom' app installed!"
	`, filepath.Clean(cwd), customDir.Path())
	removeScript := fmt.Sprintf(`#!/bin/sh
	set -e
	echo "Removing 'custom' app ..."
	export PATH=/bin:/usr/bin
	cd '%s' && pwd
	CUSTOM_DIR='%s'
	rm -f $CUSTOM_DIR/custom.file
	rmdir $CUSTOM_DIR
	echo "'Custom' app removed!"
)`, filepath.Clean(cwd), customDir.Path())

	dir := fs.NewDir(t, "test-install-custom",
		fs.WithDir(buildCtx,
			fs.WithFile("Dockerfile", fmt.Sprintf(`
			FROM %s
			COPY . /egress
			CMD ["echo", "'custom' app successfully built!"]
			`, fixtures.AlpineImage)),
			fs.WithFile(".dockerignore", `
			Dockerfile
			.dockerignore
			`, fs.WithMode(0o644)),
			fs.WithFile("LICENSE", "", fs.WithMode(0o644)),
			fs.WithFile("README.md", "", fs.WithMode(0o644)),
			fs.WithFile("install", installScript, fs.WithMode(0o755)),
			fs.WithFile("uninstall", removeScript, fs.WithMode(0o755)),
		),
	)
	defer dir.Remove()

	appBase := fs.NewDir(t, "docker-app-base",
		fs.WithDir("bin",
			fs.WithMode(os.FileMode(0o755))),
		fs.WithDir("pkg",
			fs.WithMode(os.FileMode(0o755))),
	)
	defer appBase.Remove()
	t.Setenv("DOCKER_APP_BASE", appBase.Path())

	installedApp := filepath.Join(appBase.Path(), "pkg", "file", dir.Path(), buildCtx)

	result := icmd.RunCmd(
		icmd.Command("docker", "app", "install", buildCtx),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)
	assert.Check(t, fileExists(installedApp))
	assert.Check(t, fileExists(customDir.Path()))
	assert.Check(t, fileExists(customFile))

	result = icmd.RunCmd(
		icmd.Command("docker", "app", "remove", buildCtx),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)
	assert.Check(t, !fileExists(installedApp))
	assert.Check(t, !fileExists(customDir.Path()))
	assert.Check(t, !fileExists(customFile))
	assert.Check(t, !fileExists(filepath.Join(appBase.Path(), "pkg", "file")))
	assert.Check(t, fileExists(filepath.Join(appBase.Path(), "pkg")))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
