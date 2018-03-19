//+build linux

package image

import (
	"bytes"
	"io"
	"io/ioutil"
	"syscall"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/fs"
	"golang.org/x/net/context"
)

func TestRunBuildResetsUidAndGidInContext(t *testing.T) {
	dest := fs.NewDir(t, "test-build-context-dest")
	defer dest.Remove()

	fakeImageBuild := func(_ context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
		assert.NilError(t, archive.Untar(context, dest.Path(), nil))

		body := new(bytes.Buffer)
		return types.ImageBuildResponse{Body: ioutil.NopCloser(body)}, nil
	}
	cli := test.NewFakeCli(&fakeClient{imageBuildFunc: fakeImageBuild})

	dir := fs.NewDir(t, "test-build-context",
		fs.WithFile("foo", "some content", fs.AsUser(65534, 65534)),
		fs.WithFile("Dockerfile", `
			FROM alpine:3.6
			COPY foo bar /
		`),
	)
	defer dir.Remove()

	options := newBuildOptions()
	options.context = dir.Path()
	options.untrusted = true

	err := runBuild(cli, options)
	assert.NilError(t, err)

	files, err := ioutil.ReadDir(dest.Path())
	assert.NilError(t, err)
	for _, fileInfo := range files {
		assert.Check(t, is.Equal(uint32(0), fileInfo.Sys().(*syscall.Stat_t).Uid))
		assert.Check(t, is.Equal(uint32(0), fileInfo.Sys().(*syscall.Stat_t).Gid))
	}
}
