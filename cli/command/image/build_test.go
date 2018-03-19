package image

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/fs"
	"golang.org/x/net/context"
)

func TestRunBuildDockerfileFromStdinWithCompress(t *testing.T) {
	dest, err := ioutil.TempDir("", "test-build-compress-dest")
	assert.NilError(t, err)
	defer os.RemoveAll(dest)

	var dockerfileName string
	fakeImageBuild := func(_ context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
		buffer := new(bytes.Buffer)
		tee := io.TeeReader(context, buffer)

		assert.NilError(t, archive.Untar(tee, dest, nil))
		dockerfileName = options.Dockerfile

		header := buffer.Bytes()[:10]
		assert.Check(t, is.Equal(archive.Gzip, archive.DetectCompression(header)))

		body := new(bytes.Buffer)
		return types.ImageBuildResponse{Body: ioutil.NopCloser(body)}, nil
	}

	cli := test.NewFakeCli(&fakeClient{imageBuildFunc: fakeImageBuild})
	dockerfile := bytes.NewBufferString(`
		FROM alpine:3.6
		COPY foo /
	`)
	cli.SetIn(command.NewInStream(ioutil.NopCloser(dockerfile)))

	dir, err := ioutil.TempDir("", "test-build-compress")
	assert.NilError(t, err)
	defer os.RemoveAll(dir)

	ioutil.WriteFile(filepath.Join(dir, "foo"), []byte("some content"), 0644)

	options := newBuildOptions()
	options.compress = true
	options.dockerfileName = "-"
	options.context = dir
	options.untrusted = true

	err = runBuild(cli, options)
	assert.NilError(t, err)

	files, err := ioutil.ReadDir(dest)
	assert.NilError(t, err)
	actual := []string{}
	for _, fileInfo := range files {
		actual = append(actual, fileInfo.Name())
	}
	sort.Strings(actual)
	assert.Check(t, is.DeepEqual([]string{dockerfileName, ".dockerignore", "foo"}, actual))
}

func TestRunBuildDockerfileOutsideContext(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("data", "data file"),
	)
	defer dir.Remove()

	// Dockerfile outside of build-context
	df := fs.NewFile(t, t.Name(),
		fs.WithContent(`
FROM FOOBAR
COPY data /data
		`),
	)
	defer df.Remove()

	dest, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(dest)

	var dockerfileName string
	fakeImageBuild := func(_ context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
		buffer := new(bytes.Buffer)
		tee := io.TeeReader(context, buffer)

		assert.NilError(t, archive.Untar(tee, dest, nil))
		dockerfileName = options.Dockerfile

		body := new(bytes.Buffer)
		return types.ImageBuildResponse{Body: ioutil.NopCloser(body)}, nil
	}

	cli := test.NewFakeCli(&fakeClient{imageBuildFunc: fakeImageBuild})

	options := newBuildOptions()
	options.context = dir.Path()
	options.dockerfileName = df.Path()
	options.untrusted = true

	err = runBuild(cli, options)
	assert.NilError(t, err)

	files, err := ioutil.ReadDir(dest)
	assert.NilError(t, err)
	var actual []string
	for _, fileInfo := range files {
		actual = append(actual, fileInfo.Name())
	}
	sort.Strings(actual)
	assert.Check(t, is.DeepEqual([]string{dockerfileName, ".dockerignore", "data"}, actual))
}

// TestRunBuildFromLocalGitHubDirNonExistingRepo tests that build contexts
// starting with `github.com/` are special-cased, and the build command attempts
// to clone the remote repo.
// TODO: test "context selection" logic directly when runBuild is refactored
// to support testing (ex: docker/cli#294)
func TestRunBuildFromGitHubSpecialCase(t *testing.T) {
	cmd := NewBuildCommand(test.NewFakeCli(nil))
	// Clone a small repo that exists so git doesn't prompt for credentials
	cmd.SetArgs([]string{"github.com/docker/for-win"})
	cmd.SetOutput(ioutil.Discard)
	err := cmd.Execute()
	assert.ErrorContains(t, err, "unable to prepare context")
	assert.ErrorContains(t, err, "docker-build-git")
}

// TestRunBuildFromLocalGitHubDirNonExistingRepo tests that a local directory
// starting with `github.com` takes precedence over the `github.com` special
// case.
func TestRunBuildFromLocalGitHubDir(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "docker-build-from-local-dir-")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpDir)

	buildDir := filepath.Join(tmpDir, "github.com", "docker", "no-such-repository")
	err = os.MkdirAll(buildDir, 0777)
	assert.NilError(t, err)
	err = ioutil.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte("FROM busybox\n"), 0644)
	assert.NilError(t, err)

	client := test.NewFakeCli(&fakeClient{})
	cmd := NewBuildCommand(client)
	cmd.SetArgs([]string{buildDir})
	cmd.SetOutput(ioutil.Discard)
	err = cmd.Execute()
	assert.NilError(t, err)
}

func TestRunBuildWithSymlinkedContext(t *testing.T) {
	dockerfile := `
FROM alpine:3.6
RUN echo hello world
`

	tmpDir := fs.NewDir(t, t.Name(),
		fs.WithDir("context",
			fs.WithFile("Dockerfile", dockerfile)),
		fs.WithSymlink("context-link", "context"))
	defer tmpDir.Remove()

	files := []string{}
	fakeImageBuild := func(_ context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
		tarReader := tar.NewReader(context)
		for {
			hdr, err := tarReader.Next()
			switch err {
			case io.EOF:
				body := new(bytes.Buffer)
				return types.ImageBuildResponse{Body: ioutil.NopCloser(body)}, nil
			case nil:
				files = append(files, hdr.Name)
			default:
				return types.ImageBuildResponse{}, err
			}
		}
	}

	cli := test.NewFakeCli(&fakeClient{imageBuildFunc: fakeImageBuild})
	options := newBuildOptions()
	options.context = tmpDir.Join("context-link")
	options.untrusted = true
	assert.NilError(t, runBuild(cli, options))

	assert.DeepEqual(t, files, []string{"Dockerfile"})
}
