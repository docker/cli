package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/output"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

const defaultArgs = "DOCKER_APP_BASE DOCKER_APP_PATH VERSION HOSTARCH HOSTOS USERGID USERHOME USERID USERNAME"

func TestInstallOne(t *testing.T) {
	const buildCtx = "one"
	const coolApp = "cool"
	const coolScript = `#!/bin/sh
	echo "Running 'cool' app $@ ..."
	exit 0
	`
	dir := fs.NewDir(t, "test-install-single-file",
		fs.WithDir(buildCtx,
			fs.WithFile("Dockerfile", fmt.Sprintf(`
			FROM %s
			ARG %s
			COPY cool /egress/%s
			CMD ["echo", "'cool' app successfully built!"]
			`, fixtures.AlpineImage, defaultArgs, coolApp)),
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
	t.Setenv("DOCKER_APP_BASE", appBase.Path())

	result := icmd.RunCmd(
		icmd.Command("docker", "app", "install", "--no-cache", "--launch", buildCtx, "--", "arg1", "arg2"),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)

	// verify coolScript is installed/executed on host
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0:  output.Prefix("Sending build context to Docker daemon"),
		13: output.Prefix("Successfully built "),
		14: output.Prefix("Image ID: "),
		15: output.Equals("'cool' app successfully built!"),
		16: output.Prefix("Container ID: "),
		17: output.Prefix("App copied to "),
		18: output.Prefix("App installed: "),
		19: output.Equals("Running 'cool' app arg1 arg2 ..."),
	})
	installedScript, err := os.ReadFile(installedLink)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(string(installedScript), coolScript))
}

func TestInstallMulti(t *testing.T) {
	const buildCtx = "multi"
	const runScript = `#!/bin/sh
	echo "Running 'multi' $@ ..."
	##	
	`
	dir := fs.NewDir(t, "test-install-multi-file",
		fs.WithDir(buildCtx,
			fs.WithFile("Dockerfile", fmt.Sprintf(`
			FROM %s
			ARG %s
			COPY . /egress
			CMD ["echo", "'multi' app successfully built!"]
			`, fixtures.AlpineImage, defaultArgs)),
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
		icmd.Command("docker", "app", "install", "--no-cache", "--launch", buildCtx, "--", "serve", "-p", "8080", "--arg1", "--arg2"),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)

	// verify runScript is installed/executed on host
	// and the symlink is created appropriately
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0:  output.Prefix("Sending build context to Docker daemon"),
		13: output.Prefix("Successfully built "),
		14: output.Prefix("Image ID: "),
		15: output.Equals("'multi' app successfully built!"),
		16: output.Prefix("Container ID: "),
		17: output.Prefix("App copied to "),
		18: output.Prefix("App installed: "),
		19: output.Equals("Running 'multi' serve -p 8080 --arg1 --arg2 ..."),
	})
	link, err := os.Readlink(installedLink)
	assert.NilError(t, err)
	runPath := filepath.Join(installedApp, "run")
	assert.Check(t, is.Equal(link, runPath))
	installedRunScript, err := os.ReadFile(runPath)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(string(installedRunScript), runScript))
	cnt, _ := countFiles(installedApp)
	assert.Check(t, is.Equal(cnt, 3))
}

func TestInstallCustom(t *testing.T) {
	const buildCtx = "custom"

	cwd, err := os.Getwd()
	assert.NilError(t, err, "failed to get cwd for test")

	installScript := fmt.Sprintf(`#!/bin/sh
	echo "Installing 'custom' app $@ ..."
	echo "$DOCKER_APP_BASE"
	cd '%s' && pwd
	echo "$PATH"
	echo "'Custom' app installed!"
	`, filepath.Clean(cwd))

	dir := fs.NewDir(t, "test-install-custom",
		fs.WithDir(buildCtx,
			fs.WithFile("Dockerfile", fmt.Sprintf(`
			FROM %s
			ARG %s
			COPY . /egress
			CMD ["echo", "'custom' app successfully built!"]
			`, fixtures.AlpineImage, defaultArgs)),
			fs.WithFile(".dockerignore", `
			Dockerfile
			.dockerignore
			`, fs.WithMode(0o644)),
			fs.WithFile("LICENSE", "", fs.WithMode(0o644)),
			fs.WithFile("README.md", "", fs.WithMode(0o644)),
			fs.WithFile("install", installScript, fs.WithMode(0o755)),
			fs.WithFile("uninstall", "", fs.WithMode(0o755)),
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

	result := icmd.RunCmd(
		icmd.Command("docker", "app", "install", "--no-cache", buildCtx, "--", "--arg1", "--arg2"),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)

	// verify installScript is executed on host
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0:  output.Prefix("Sending build context to Docker daemon"),
		13: output.Prefix("Successfully built "),
		14: output.Prefix("Image ID: "),
		15: output.Equals("'custom' app successfully built!"),
		16: output.Prefix("Container ID: "),
		17: output.Prefix("App copied to "),
		18: output.Equals("Installing 'custom' app --arg1 --arg2 ..."),
		19: output.Equals(appBase.Path()),
		20: output.Equals(cwd),
		21: output.Equals(os.Getenv("PATH")),
		22: output.Equals("'Custom' app installed!"),
		23: output.Equals("App installer ran successfully"),
	})
}

func TestInstallCustomDestination(t *testing.T) {
	const buildCtx = "service"

	deployScript := `#!/bin/sh
	echo "deploying 'service' $@ ..."
	echo "'service' deployed!"
	`

	egress := "/pkg/releases/v1.0.0"
	dir := fs.NewDir(t, "test-install-deploy",
		fs.WithDir(buildCtx,
			fs.WithFile("Dockerfile", fmt.Sprintf(`
			FROM %s
			ARG %s
			COPY . %s
			CMD ["echo", "'service' successfully built!"]
			`, fixtures.AlpineImage, defaultArgs, egress)),
			fs.WithFile("config", "", fs.WithMode(0o644)),
			fs.WithFile("install", deployScript, fs.WithMode(0o755)),
		),
	)
	defer dir.Remove()

	appBase := fs.NewDir(t, "docker-app-base")
	defer appBase.Remove()
	t.Setenv("DOCKER_APP_BASE", appBase.Path())

	destBase := fs.NewDir(t, "custom-destination")
	defer destBase.Remove()
	dest := filepath.Join(destBase.Path(), "stage")
	config := filepath.Join(dest, "config")

	result := icmd.RunCmd(
		icmd.Command("docker", "app", "install", "--no-cache", "--destination", dest, "--egress", egress, buildCtx, "--", "arg1", "arg2"),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)

	// verify deployScript is installed/executed on host
	// and "config" file is copied to custom destination
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0:  output.Prefix("Sending build context to Docker daemon"),
		13: output.Prefix("Successfully built "),
		14: output.Prefix("Image ID: "),
		15: output.Equals("'service' successfully built!"),
		16: output.Prefix("Container ID: "),
		17: output.Prefix("App copied to "),
		18: output.Equals("deploying 'service' arg1 arg2 ..."),
		19: output.Equals("'service' deployed!"),
	})
	assert.Check(t, fileExists(config))
}

func withWorkingDir(dir *fs.Dir) func(*icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		cmd.Dir = dir.Path()
	}
}

func countFiles(dir string) (int, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	cnt := 0
	for _, file := range files {
		if !file.IsDir() {
			cnt++
		}
	}
	return cnt, nil
}
