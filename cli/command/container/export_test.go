package container

import (
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestContainerExportOutputToFile(t *testing.T) {
	dir := fs.NewDir(t, "export-test")
	defer dir.Remove()

	cli := test.NewFakeCli(&fakeClient{
		containerExportFunc: func(container string) (client.ContainerExportResult, error) {
			// FIXME(thaJeztah): how to mock this?
			return mockContainerExportResult("bar"), nil
		},
	})
	cmd := newExportCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"-o", dir.Join("foo"), "container"})
	assert.NilError(t, cmd.Execute())

	expected := fs.Expected(t,
		fs.WithFile("foo", "bar", fs.MatchAnyFileMode),
	)

	assert.Assert(t, fs.Equal(dir.Path(), expected))
}

func TestContainerExportOutputToIrregularFile(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerExportFunc: func(container string) (client.ContainerExportResult, error) {
			// FIXME(thaJeztah): how to mock this?
			return mockContainerExportResult("foo"), nil
		},
	})
	cmd := newExportCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"-o", "/dev/random", "container"})

	const expected = `failed to export container: cannot write to a character device file`
	assert.Error(t, cmd.Execute(), expected)
}
