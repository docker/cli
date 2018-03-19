package image

import (
	"fmt"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/output"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/icmd"
)

func TestBuildFromContextDirectoryWithTag(t *testing.T) {
	dir := fs.NewDir(t, "test-build-context-dir",
		fs.WithFile("run", "echo running", fs.WithMode(0755)),
		fs.WithDir("data", fs.WithFile("one", "1111")),
		fs.WithFile("Dockerfile", fmt.Sprintf(`
	FROM	%s
	COPY	run /usr/bin/run
	RUN		run
	COPY	data /data
		`, fixtures.AlpineImage)))
	defer dir.Remove()

	result := icmd.RunCmd(
		icmd.Command("docker", "build", "-t", "myimage", "."),
		withWorkingDir(dir))

	result.Assert(t, icmd.Expected{Err: icmd.None})
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0:  output.Prefix("Sending build context to Docker daemon"),
		1:  output.Equals("Step 1/4 : FROM\tregistry:5000/alpine:3.6"),
		3:  output.Equals("Step 2/4 : COPY\trun /usr/bin/run"),
		5:  output.Equals("Step 3/4 : RUN\t\trun"),
		7:  output.Equals("running"),
		8:  output.Prefix("Removing intermediate container "),
		10: output.Equals("Step 4/4 : COPY\tdata /data"),
		12: output.Prefix("Successfully built "),
		13: output.Equals("Successfully tagged myimage:latest"),
	})
}

func TestTrustedBuild(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	image1 := fixtures.CreateMaskedTrustedRemoteImage(t, registryPrefix, "trust-build1", "latest")
	image2 := fixtures.CreateMaskedTrustedRemoteImage(t, registryPrefix, "trust-build2", "latest")

	buildDir := fs.NewDir(t, "test-trusted-build-context-dir",
		fs.WithFile("Dockerfile", fmt.Sprintf(`
	FROM %s as build-base
	RUN echo ok > /foo
	FROM %s
	COPY --from=build-base foo bar
		`, image1, image2)))
	defer buildDir.Remove()

	result := icmd.RunCmd(
		icmd.Command("docker", "build", "-t", "myimage", "."),
		withWorkingDir(buildDir),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)

	result.Assert(t, icmd.Expected{
		Out: fmt.Sprintf("FROM %s@sha", image1[:len(image1)-7]),
		Err: fmt.Sprintf("Tagging %s@sha", image1[:len(image1)-7]),
	})
	result.Assert(t, icmd.Expected{
		Out: fmt.Sprintf("FROM %s@sha", image2[:len(image2)-7]),
	})
}

func TestTrustedBuildUntrustedImage(t *testing.T) {
	dir := fixtures.SetupConfigFile(t)
	defer dir.Remove()
	buildDir := fs.NewDir(t, "test-trusted-build-context-dir",
		fs.WithFile("Dockerfile", fmt.Sprintf(`
	FROM %s
	RUN []
		`, fixtures.AlpineImage)))
	defer buildDir.Remove()

	result := icmd.RunCmd(
		icmd.Command("docker", "build", "-t", "myimage", "."),
		withWorkingDir(buildDir),
		fixtures.WithConfig(dir.Path()),
		fixtures.WithTrust,
		fixtures.WithNotary,
	)

	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "does not have trust data for",
	})
}

func withWorkingDir(dir *fs.Dir) func(*icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		cmd.Dir = dir.Path()
	}
}
