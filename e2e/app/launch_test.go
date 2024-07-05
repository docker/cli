package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/output"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

func TestLaunchOne(t *testing.T) {
	const buildCtx = "one"
	const coolApp = "cool"

	cwd, err := os.Getwd()
	assert.NilError(t, err, "failed to get cwd for test")

	testMsg := fmt.Sprintf("It is %v", time.Now())

	coolScript := fmt.Sprintf(`#!/bin/sh
	echo "Running 'cool' app ..."
	cd '%s' && pwd
	echo "$PATH"
	echo "%s"
	`, filepath.Clean(cwd), testMsg)

	dir := fs.NewDir(t, "test-launch-single-file",
		fs.WithDir(buildCtx,
			fs.WithFile("Dockerfile", fmt.Sprintf(`
			FROM %s
			ARG HOSTOS HOSTARCH
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
	t.Setenv("DOCKER_APP_BASE", appBase.Path())

	result := icmd.RunCmd(
		icmd.Command("docker", "app", "launch", "--no-cache", buildCtx),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)

	// verify coolScript is executed on host
	// by comparing the cwd, PATH, and the random test message
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0:  output.Prefix("Sending build context to Docker daemon"),
		18: output.Equals("Running 'cool' app ..."),
		19: output.Equals(cwd),
		20: output.Equals(os.Getenv("PATH")),
		21: output.Equals(testMsg),
	})
}

func TestLaunchMulti(t *testing.T) {
	const buildCtx = "multi"

	cwd, err := os.Getwd()
	assert.NilError(t, err, "failed to get cwd for test")

	testMsg := fmt.Sprintf("It is %v", time.Now())

	runScript := fmt.Sprintf(`#!/bin/sh
	echo "Running 'multi' ..."
	cd '%s' && pwd
	echo "$PATH"
	echo "%s"
	##	
	`, filepath.Clean(cwd), testMsg)

	dir := fs.NewDir(t, "test-launch-multi-file",
		fs.WithDir(buildCtx,
			fs.WithFile("Dockerfile", fmt.Sprintf(`
			FROM %s
			ARG HOSTOS HOSTARCH
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
	t.Setenv("DOCKER_APP_BASE", appBase.Path())

	result := icmd.RunCmd(
		icmd.Command("docker", "app", "launch", "--no-cache", buildCtx),
		withWorkingDir(dir),
	)
	result.Assert(t, icmd.Success)

	// verify runScript is executed on host
	// by comparing the cwd, PATH, and the random test message
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0:  output.Prefix("Sending build context to Docker daemon"),
		18: output.Equals("Running 'multi' ..."),
		19: output.Equals(cwd),
		20: output.Equals(os.Getenv("PATH")),
		21: output.Equals(testMsg),
	})
}
