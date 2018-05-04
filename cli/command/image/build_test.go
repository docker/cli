package image

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
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
	"github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/skip"
)

func TestRunBuildDockerfileFromStdinWithCompress(t *testing.T) {
	buffer := new(bytes.Buffer)
	fakeBuild := newFakeBuild()
	fakeImageBuild := func(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
		tee := io.TeeReader(context, buffer)
		gzipReader, err := gzip.NewReader(tee)
		assert.NilError(t, err)
		return fakeBuild.build(ctx, gzipReader, options)
	}

	cli := test.NewFakeCli(&fakeClient{imageBuildFunc: fakeImageBuild})
	dockerfile := bytes.NewBufferString(`
		FROM alpine:3.6
		COPY foo /
	`)
	cli.SetIn(command.NewInStream(ioutil.NopCloser(dockerfile)))

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("foo", "some content"))
	defer dir.Remove()

	options := newBuildOptions()
	options.compress = true
	options.dockerfileName = "-"
	options.context = dir.Path()
	options.untrusted = true
	assert.NilError(t, runBuild(cli, options))

	expected := []string{fakeBuild.options.Dockerfile, ".dockerignore", "foo"}
	assert.DeepEqual(t, expected, fakeBuild.filenames(t))

	header := buffer.Bytes()[:10]
	assert.Equal(t, archive.Gzip, archive.DetectCompression(header))
}

func TestRunBuildResetsUidAndGidInContext(t *testing.T) {
	skip.If(t, os.Getuid() != 0, "root is required to chown files")
	fakeBuild := newFakeBuild()
	cli := test.NewFakeCli(&fakeClient{imageBuildFunc: fakeBuild.build})

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
	assert.NilError(t, runBuild(cli, options))

	headers := fakeBuild.headers(t)
	expected := []*tar.Header{
		{Name: "Dockerfile"},
		{Name: "foo"},
	}
	var cmpTarHeaderNameAndOwner = cmp.Comparer(func(x, y tar.Header) bool {
		return x.Name == y.Name && x.Uid == y.Uid && x.Gid == y.Gid
	})
	assert.DeepEqual(t, expected, headers, cmpTarHeaderNameAndOwner)
}

func TestRunBuildDockerfileOutsideContext(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("data", "data file"))
	defer dir.Remove()

	// Dockerfile outside of build-context
	df := fs.NewFile(t, t.Name(),
		fs.WithContent(`
FROM FOOBAR
COPY data /data
		`),
	)
	defer df.Remove()

	fakeBuild := newFakeBuild()
	cli := test.NewFakeCli(&fakeClient{imageBuildFunc: fakeBuild.build})

	options := newBuildOptions()
	options.context = dir.Path()
	options.dockerfileName = df.Path()
	options.untrusted = true
	assert.NilError(t, runBuild(cli, options))

	expected := []string{fakeBuild.options.Dockerfile, ".dockerignore", "data"}
	assert.DeepEqual(t, expected, fakeBuild.filenames(t))
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

	fakeBuild := newFakeBuild()
	cli := test.NewFakeCli(&fakeClient{imageBuildFunc: fakeBuild.build})
	options := newBuildOptions()
	options.context = tmpDir.Join("context-link")
	options.untrusted = true
	assert.NilError(t, runBuild(cli, options))

	assert.DeepEqual(t, fakeBuild.filenames(t), []string{"Dockerfile"})
}

type fakeBuild struct {
	context *tar.Reader
	options types.ImageBuildOptions
}

func newFakeBuild() *fakeBuild {
	return &fakeBuild{}
}

func (f *fakeBuild) build(_ context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	f.context = tar.NewReader(context)
	f.options = options
	body := new(bytes.Buffer)
	return types.ImageBuildResponse{Body: ioutil.NopCloser(body)}, nil
}

func (f *fakeBuild) headers(t *testing.T) []*tar.Header {
	t.Helper()
	headers := []*tar.Header{}
	for {
		hdr, err := f.context.Next()
		switch err {
		case io.EOF:
			return headers
		case nil:
			headers = append(headers, hdr)
		default:
			assert.NilError(t, err)
		}
	}
}

func (f *fakeBuild) filenames(t *testing.T) []string {
	t.Helper()
	names := []string{}
	for _, header := range f.headers(t) {
		names = append(names, header.Name)
	}
	sort.Strings(names)
	return names
}
