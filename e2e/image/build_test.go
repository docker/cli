package image

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/pkg/errors"
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
	assertBuildOutput(t, result.Stdout(), map[int]lineCompare{
		0:  prefix("Sending build context to Docker daemon"),
		1:  equals("Step 1/4 : FROM\tregistry:5000/alpine:3.6"),
		3:  equals("Step 2/4 : COPY\trun /usr/bin/run"),
		5:  equals("Step 3/4 : RUN\t\trun"),
		7:  equals("running"),
		8:  prefix("Removing intermediate container "),
		10: equals("Step 4/4 : COPY\tdata /data"),
		12: prefix("Successfully built "),
		13: equals("Successfully tagged myimage:latest"),
	})
}

func withWorkingDir(dir *fs.Dir) func(*icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		cmd.Dir = dir.Path()
	}
}

func assertBuildOutput(t *testing.T, actual string, expectedLines map[int]lineCompare) {
	for i, line := range strings.Split(actual, "\n") {
		cmp, ok := expectedLines[i]
		if !ok {
			continue
		}
		if err := cmp(line); err != nil {
			t.Errorf("line %d: %s", i, err)
		}
	}
	if t.Failed() {
		t.Log(actual)
	}
}

type lineCompare func(string) error

func prefix(expected string) func(string) error {
	return func(actual string) error {
		if strings.HasPrefix(actual, expected) {
			return nil
		}
		return errors.Errorf("expected %s to start with %s", actual, expected)
	}
}

func equals(expected string) func(string) error {
	return func(actual string) error {
		if expected == actual {
			return nil
		}
		return errors.Errorf("got %s, expected %s", actual, expected)
	}
}
