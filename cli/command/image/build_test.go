package image

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestRunBuildDockerfileFromStdinWithCompress(t *testing.T) {
	dest, err := ioutil.TempDir("", "test-build-compress-dest")
	require.NoError(t, err)
	defer os.RemoveAll(dest)

	var dockerfileName string
	fakeImageBuild := func(_ context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
		buffer := new(bytes.Buffer)
		tee := io.TeeReader(context, buffer)

		assert.NoError(t, archive.Untar(tee, dest, nil))
		dockerfileName = options.Dockerfile

		header := buffer.Bytes()[:10]
		assert.Equal(t, archive.Gzip, archive.DetectCompression(header))

		body := new(bytes.Buffer)
		return types.ImageBuildResponse{Body: ioutil.NopCloser(body)}, nil
	}

	cli := test.NewFakeCliWithOutput(&fakeClient{imageBuildFunc: fakeImageBuild}, ioutil.Discard)
	dockerfile := bytes.NewBufferString(`
		FROM alpine:3.6
		COPY foo /
	`)
	cli.SetIn(command.NewInStream(ioutil.NopCloser(dockerfile)))

	dir, err := ioutil.TempDir("", "test-build-compress")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	ioutil.WriteFile(filepath.Join(dir, "foo"), []byte("some content"), 0644)

	options := newBuildOptions()
	options.compress = true
	options.dockerfileName = "-"
	options.context = dir

	err = runBuild(cli, options)
	require.NoError(t, err)

	files, err := ioutil.ReadDir(dest)
	require.NoError(t, err)
	actual := []string{}
	for _, fileInfo := range files {
		actual = append(actual, fileInfo.Name())
	}
	sort.Strings(actual)
	assert.Equal(t, []string{dockerfileName, ".dockerignore", "foo"}, actual)
}
