package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/docker/cli/internal/test/environment"
	"github.com/docker/cli/internal/test/output"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

func TestBuildFromContextDirectoryWithTag(t *testing.T) {
	t.Setenv("DOCKER_BUILDKIT", "0")

	dir := fs.NewDir(t, "test-build-context-dir",
		fs.WithFile("run", "echo running", fs.WithMode(0o755)),
		fs.WithDir("data", fs.WithFile("one", "1111")),
		fs.WithFile("Dockerfile", fmt.Sprintf(`
	FROM %s
	COPY run /usr/bin/run
	RUN run
	COPY data /data
		`, fixtures.AlpineImage)))
	defer dir.Remove()

	result := icmd.RunCmd(
		icmd.Command("docker", "build", "-t", "myimage", "."),
		withWorkingDir(dir))
	defer icmd.RunCommand("docker", "image", "rm", "myimage")

	const buildkitDisabledWarning = `DEPRECATED: The legacy builder is deprecated and will be removed in a future release.
            BuildKit is currently disabled; enable it by removing the DOCKER_BUILDKIT=0
            environment-variable.
`

	result.Assert(t, icmd.Expected{Err: buildkitDisabledWarning})
	output.Assert(t, result.Stdout(), map[int]func(string) error{
		0: output.Prefix("Sending build context to Docker daemon"),
		1: output.Suffix("Step 1/4 : FROM registry:5000/alpine:frozen"),
		3: output.Suffix("Step 2/4 : COPY run /usr/bin/run"),
		5: output.Suffix("Step 3/4 : RUN run"),
		7: output.Suffix("running"),
		// TODO(krissetto): ugly, remove when no longer testing against moby 24. see https://github.com/moby/moby/pull/46270
		8: func(s string) error {
			err := output.Contains("Removed intermediate container")(s) // moby >= v25
			if err == nil {
				return nil
			}
			return output.Contains("Removing intermediate container")(s) // moby < v25
		},
		10: output.Suffix("Step 4/4 : COPY data /data"),
		12: output.Contains("Successfully built "),
		13: output.Suffix("Successfully tagged myimage:latest"),
	})
}

func TestBuildIidFileSquash(t *testing.T) {
	t.Skip("Not implemented with containerd")
	environment.SkipIfNotExperimentalDaemon(t)
	t.Setenv("DOCKER_BUILDKIT", "0")

	dir := fs.NewDir(t, "test-iidfile-squash")
	defer dir.Remove()
	iidfile := filepath.Join(dir.Path(), "idsquash")
	buildDir := fs.NewDir(t, "test-iidfile-squash-build",
		fs.WithFile("Dockerfile", fmt.Sprintf(`
	FROM %s
	ENV FOO=FOO
	ENV BAR=BAR
	RUN touch /fiip
	RUN touch /foop`, fixtures.AlpineImage)),
	)
	defer buildDir.Remove()

	imageTag := "testbuildiidfilesquash"
	result := icmd.RunCmd(
		icmd.Command("docker", "build", "--iidfile", iidfile, "--squash", "-t", imageTag, "."),
		withWorkingDir(buildDir),
	)
	result.Assert(t, icmd.Success)
	id, err := os.ReadFile(iidfile)
	assert.NilError(t, err)
	result = icmd.RunCommand("docker", "image", "inspect", "-f", "{{.Id}}", imageTag)
	result.Assert(t, icmd.Success)
	assert.Check(t, is.Equal(string(id), strings.TrimSpace(result.Combined())))
}

func withWorkingDir(dir *fs.Dir) func(*icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		cmd.Dir = dir.Path()
	}
}
