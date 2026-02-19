package container

import (
	"context"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/go-archive"
	"github.com/moby/go-archive/compression"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"
)

func TestRunCopyWithInvalidArguments(t *testing.T) {
	testcases := []struct {
		doc         string
		options     copyOptions
		expectedErr string
	}{
		{
			doc: "copy between container",
			options: copyOptions{
				source:      "first:/path",
				destination: "second:/path",
			},
			expectedErr: "copying between containers is not supported",
		},
		{
			doc: "copy without a container",
			options: copyOptions{
				source:      "./source",
				destination: "./dest",
			},
			expectedErr: "must specify at least one container source",
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			err := runCopy(context.TODO(), test.NewFakeCli(nil), testcase.options)
			assert.Error(t, err, testcase.expectedErr)
		})
	}
}

func TestRunCopyFromContainerToStdout(t *testing.T) {
	tarContent := "the tar content"

	cli := test.NewFakeCli(&fakeClient{
		containerCopyFromFunc: func(ctr, srcPath string) (client.CopyFromContainerResult, error) {
			assert.Check(t, is.Equal("container", ctr))
			return client.CopyFromContainerResult{
				Content: io.NopCloser(strings.NewReader(tarContent)),
			}, nil
		},
	})
	err := runCopy(context.TODO(), cli, copyOptions{
		source:      "container:/path",
		destination: "-",
	})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(tarContent, cli.OutBuffer().String()))
	assert.Check(t, is.Equal("", cli.ErrBuffer().String()))
}

func TestRunCopyFromContainerToFilesystem(t *testing.T) {
	srcDir := fs.NewDir(t, "cp-test",
		fs.WithFile("file1", "content\n"))

	destDir := fs.NewDir(t, "cp-test")

	cli := test.NewFakeCli(&fakeClient{
		containerCopyFromFunc: func(ctr, srcPath string) (client.CopyFromContainerResult, error) {
			assert.Check(t, is.Equal("container", ctr))
			readCloser, err := archive.Tar(srcDir.Path(), compression.None)
			return client.CopyFromContainerResult{
				Content: readCloser,
			}, err
		},
	})
	err := runCopy(context.TODO(), cli, copyOptions{
		source:      "container:/path",
		destination: destDir.Path(),
		quiet:       true,
	})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("", cli.OutBuffer().String()))
	assert.Check(t, is.Equal("", cli.ErrBuffer().String()))

	content, err := os.ReadFile(destDir.Join("file1"))
	assert.NilError(t, err)
	assert.Check(t, is.Equal("content\n", string(content)))
}

func TestRunCopyFromContainerToFilesystemMissingDestinationDirectory(t *testing.T) {
	destDir := fs.NewDir(t, "cp-test",
		fs.WithFile("file1", "content\n"))
	defer destDir.Remove()

	cli := test.NewFakeCli(&fakeClient{
		containerCopyFromFunc: func(ctr, srcPath string) (client.CopyFromContainerResult, error) {
			assert.Check(t, is.Equal("container", ctr))
			readCloser, err := archive.TarWithOptions(destDir.Path(), &archive.TarOptions{})
			return client.CopyFromContainerResult{
				Content: readCloser,
			}, err
		},
	})
	err := runCopy(context.TODO(), cli, copyOptions{
		source:      "container:/path",
		destination: destDir.Join("missing", "foo"),
	})
	assert.ErrorContains(t, err, destDir.Join("missing"))
}

func TestRunCopyToContainerFromFileWithTrailingSlash(t *testing.T) {
	srcFile := fs.NewFile(t, t.Name())
	defer srcFile.Remove()

	cli := test.NewFakeCli(&fakeClient{})
	err := runCopy(context.TODO(), cli, copyOptions{
		source:      srcFile.Path() + string(os.PathSeparator),
		destination: "container:/path",
	})

	expectedError := "not a directory"
	if runtime.GOOS == "windows" {
		expectedError = "The filename, directory name, or volume label syntax is incorrect"
	}
	assert.ErrorContains(t, err, expectedError)
}

func TestRunCopyToContainerSourceDoesNotExist(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{})
	err := runCopy(context.TODO(), cli, copyOptions{
		source:      "/does/not/exist",
		destination: "container:/path",
	})
	expected := "no such file or directory"
	if runtime.GOOS == "windows" {
		expected = "cannot find the file specified"
	}
	assert.ErrorContains(t, err, expected)
}

func TestSplitCpArg(t *testing.T) {
	testcases := []struct {
		doc               string
		path              string
		os                string
		expectedContainer string
		expectedPath      string
	}{
		{
			doc:          "absolute path with colon",
			os:           "unix",
			path:         "/abs/path:withcolon",
			expectedPath: "/abs/path:withcolon",
		},
		{
			doc:          "relative path with colon",
			path:         "./relative:path",
			expectedPath: "./relative:path",
		},
		{
			doc:          "absolute path with drive",
			os:           "windows",
			path:         `d:\abs\path`,
			expectedPath: `d:\abs\path`,
		},
		{
			doc:          "no separator",
			path:         "relative/path",
			expectedPath: "relative/path",
		},
		{
			doc:               "with separator",
			path:              "container:/opt/foo",
			expectedPath:      "/opt/foo",
			expectedContainer: "container",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.doc, func(t *testing.T) {
			if tc.os == "windows" && runtime.GOOS != "windows" {
				t.Skip("skipping windows test on non-windows platform")
			}
			if tc.os == "unix" && runtime.GOOS == "windows" {
				t.Skip("skipping unix test on windows")
			}

			ctr, path := splitCpArg(tc.path)
			assert.Check(t, is.Equal(tc.expectedContainer, ctr))
			assert.Check(t, is.Equal(tc.expectedPath, path))
		})
	}
}

func TestRunCopyFromContainerToFilesystemIrregularDestination(t *testing.T) {
	cli := test.NewFakeCli(nil)
	err := runCopy(context.TODO(), cli, copyOptions{
		source:      "container:/dev/null",
		destination: "/dev/random",
	})
	assert.Assert(t, err != nil)
	expected := `"/dev/random" must be a directory or a regular file`
	assert.ErrorContains(t, err, expected)
}

func TestCopyFromContainerReportsFileSize(t *testing.T) {
	// The file content is "hello" (5 bytes), but the TAR archive wrapping
	// it is much larger due to headers and padding. The success message
	// should report the actual file size (5B), not the TAR stream size.
	srcDir := fs.NewDir(t, "cp-test-from",
		fs.WithFile("file1", "hello"))

	destDir := fs.NewDir(t, "cp-test-from-dest")

	const fileSize int64 = 5
	fakeCli := test.NewFakeCli(&fakeClient{
		containerCopyFromFunc: func(ctr, srcPath string) (client.CopyFromContainerResult, error) {
			readCloser, err := archive.Tar(srcDir.Path(), compression.None)
			return client.CopyFromContainerResult{
				Content: readCloser,
				Stat: container.PathStat{
					Name: "file1",
					Size: fileSize,
				},
			}, err
		},
	})
	err := runCopy(context.TODO(), fakeCli, copyOptions{
		source:      "container:/file1",
		destination: destDir.Path(),
	})
	assert.NilError(t, err)
	errOut := fakeCli.ErrBuffer().String()
	assert.Check(t, is.Contains(errOut, "Successfully copied 5B"))
	assert.Check(t, is.Contains(errOut, "(transferred"))
}

func TestCopyToContainerReportsFileSize(t *testing.T) {
	// Create a temp file with known content ("hello" = 5 bytes).
	// The TAR archive sent to the container is larger, but the success
	// message should report the actual content size.
	srcFile := fs.NewFile(t, "cp-test-to", fs.WithContent("hello"))

	fakeCli := test.NewFakeCli(&fakeClient{
		containerStatPathFunc: func(containerID, path string) (client.ContainerStatPathResult, error) {
			return client.ContainerStatPathResult{
				Stat: container.PathStat{
					Name: "tmp",
					Mode: os.ModeDir | 0o755,
				},
			}, nil
		},
		containerCopyToFunc: func(containerID string, options client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
			_, _ = io.Copy(io.Discard, options.Content)
			return client.CopyToContainerResult{}, nil
		},
	})
	err := runCopy(context.TODO(), fakeCli, copyOptions{
		source:      srcFile.Path(),
		destination: "container:/tmp",
	})
	assert.NilError(t, err)
	errOut := fakeCli.ErrBuffer().String()
	assert.Check(t, is.Contains(errOut, "Successfully copied 5B"))
	assert.Check(t, is.Contains(errOut, "(transferred"))
}

func TestCopyToContainerReportsEmptyFileSize(t *testing.T) {
	srcFile := fs.NewFile(t, "cp-test-empty", fs.WithContent(""))

	fakeCli := test.NewFakeCli(&fakeClient{
		containerStatPathFunc: func(containerID, path string) (client.ContainerStatPathResult, error) {
			return client.ContainerStatPathResult{
				Stat: container.PathStat{
					Name: "tmp",
					Mode: os.ModeDir | 0o755,
				},
			}, nil
		},
		containerCopyToFunc: func(containerID string, options client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
			_, _ = io.Copy(io.Discard, options.Content)
			return client.CopyToContainerResult{}, nil
		},
	})
	err := runCopy(context.TODO(), fakeCli, copyOptions{
		source:      srcFile.Path(),
		destination: "container:/tmp",
	})
	assert.NilError(t, err)
	errOut := fakeCli.ErrBuffer().String()
	assert.Check(t, is.Contains(errOut, "Successfully copied 0B"))
	assert.Check(t, is.Contains(errOut, "(transferred"))
}

func TestCopyToContainerReportsDirectorySize(t *testing.T) {
	// Create a temp directory with files "aaa" (3 bytes) + "bbb" (3 bytes) = 6 bytes.
	// The TAR archive is much larger, but the success message should report 6B.
	srcDir := fs.NewDir(t, "cp-test-dir",
		fs.WithFile("aaa", "aaa"),
		fs.WithFile("bbb", "bbb"),
	)

	fakeCli := test.NewFakeCli(&fakeClient{
		containerStatPathFunc: func(containerID, path string) (client.ContainerStatPathResult, error) {
			return client.ContainerStatPathResult{
				Stat: container.PathStat{
					Name: "tmp",
					Mode: os.ModeDir | 0o755,
				},
			}, nil
		},
		containerCopyToFunc: func(containerID string, options client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
			_, _ = io.Copy(io.Discard, options.Content)
			return client.CopyToContainerResult{}, nil
		},
	})
	err := runCopy(context.TODO(), fakeCli, copyOptions{
		source:      srcDir.Path() + string(os.PathSeparator),
		destination: "container:/tmp",
	})
	assert.NilError(t, err)
	errOut := fakeCli.ErrBuffer().String()
	assert.Check(t, is.Contains(errOut, "Successfully copied 6B"))
	assert.Check(t, is.Contains(errOut, "(transferred"))
}

func TestCopyFromContainerReportsDirectorySize(t *testing.T) {
	// When copying a directory from a container, cpRes.Stat.Mode.IsDir() is true,
	// so reportedSize falls back to copiedSize (the tar stream bytes).
	srcDir := fs.NewDir(t, "cp-test-fromdir",
		fs.WithFile("file1", "hello"))

	destDir := fs.NewDir(t, "cp-test-fromdir-dest")

	fakeCli := test.NewFakeCli(&fakeClient{
		containerCopyFromFunc: func(ctr, srcPath string) (client.CopyFromContainerResult, error) {
			readCloser, err := archive.Tar(srcDir.Path(), compression.None)
			return client.CopyFromContainerResult{
				Content: readCloser,
				Stat: container.PathStat{
					Name: "mydir",
					Mode: os.ModeDir | 0o755,
				},
			}, err
		},
	})
	err := runCopy(context.TODO(), fakeCli, copyOptions{
		source:      "container:/mydir",
		destination: destDir.Path(),
	})
	assert.NilError(t, err)
	errOut := fakeCli.ErrBuffer().String()
	assert.Check(t, is.Contains(errOut, "Successfully copied"))
	// For directories from container, content size is unknown so
	// reportedSize == copiedSize and "(transferred ...)" is omitted.
	assert.Check(t, !strings.Contains(errOut, "(transferred"))
}
