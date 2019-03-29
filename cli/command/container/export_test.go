package container

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestContainerExportOutputToFile(t *testing.T) {
	dir := fs.NewDir(t, "export-test")
	defer dir.Remove()

	cli := test.NewFakeCli(&fakeClient{
		containerExportFunc: func(container string) (io.ReadCloser, error) {
			return ioutil.NopCloser(strings.NewReader("bar")), nil
		},
	})
	cmd := NewExportCommand(cli)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"-o", dir.Join("foo"), "container"})
	assert.NilError(t, cmd.Execute())

	expected := fs.Expected(t,
		fs.WithFile("foo", "bar", fs.MatchAnyFileMode),
	)

	assert.Assert(t, fs.Equal(dir.Path(), expected))
}

func TestContainerExportOutputToIrregularFile(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerExportFunc: func(container string) (io.ReadCloser, error) {
			return ioutil.NopCloser(strings.NewReader("foo")), nil
		},
	})
	cmd := NewExportCommand(cli)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"-o", "/dev/random", "container"})

	err := cmd.Execute()
	assert.Assert(t, err != nil)
	expected := `"/dev/random" must be a directory or a regular file`
	assert.ErrorContains(t, err, expected)
}
