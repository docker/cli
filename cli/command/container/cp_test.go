package container

import (
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
	"gotest.tools/skip"
)

func TestSeparateCopyCommands(t *testing.T) {
	var testcases = []struct {
		doc         string
		options     copyOptions
		expectedErr string
		direction   copyDirection
		args        []string
	}{
		{
			doc: "copy between container",
			expectedErr: "copying between containers is not supported",
			direction:   acrossContainers,
			args: []string {"first:/path", "first:/path"},
		},
		{
			doc: "copy without container",
			expectedErr: "invalid use of cp command\n see 'docker cp --help'",
			direction:   0,
			args: []string {"/path", "/path"},
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			err := separateCopyCommands(test.NewFakeCli(nil), testcase.options, testcase.args)
			assert.Error(t, err, testcase.expectedErr)
		})
	}
}

func TestGetCpDirection(t *testing.T) {
	var testcases = []struct {
		doc            string
		source         containerWithPath
		destination    containerWithPath
		expectedResult copyDirection
	}{
		{
			doc: "container to container",
			source:      splitCpArg("first:/path"),
			destination: splitCpArg("second:/path"),
			expectedResult: acrossContainers,
		},
		{
			doc: "source to container",
			source:      splitCpArg("/path"),
			destination: splitCpArg("second:/path"),
			expectedResult: acrossContainers,
		},
		{
			doc: "container to source",
			source:      splitCpArg("first:/path"),
			destination: splitCpArg("/path"),
			expectedResult: acrossContainers,
		},
		{
			doc: "source to source",
			source:      splitCpArg("/path"),
			destination: splitCpArg("/path"),
			expectedResult: acrossContainers,
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			direction := getCpDirection(testcase.source, testcase.destination)
			assert.Equal(t, direction, testcase.expectedResult)
		})
	}
}

func TestRunCopyWithInvalidArguments(t *testing.T) {
	var testcases = []struct {
		doc         string
		options     copyOptions
		expectedErr string
		direction   copyDirection
	}{
		{
			doc: "copy between containers",
			options: copyOptions{
				source:      splitCpArg("first:/path"),
				destination: splitCpArg("second:/path"),
			},
			expectedErr: "copying between containers is not supported",
			direction:   acrossContainers,
		},
		{
			doc: "copy without a container",
			options: copyOptions{
				source:      splitCpArg("./source"),
				destination: splitCpArg("./dest"),
			},
			expectedErr: "must specify at least one container source",
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			err := runCopy(test.NewFakeCli(nil), testcase.options, testcase.direction)
			assert.Error(t, err, testcase.expectedErr)
		})
	}
}

func TestRunCopyFromContainerToStdout(t *testing.T) {
	tarContent := "the tar content"

	fakeClient := &fakeClient{
		containerCopyFromFunc: func(container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
			assert.Check(t, is.Equal("container", container))
			return ioutil.NopCloser(strings.NewReader(tarContent)), types.ContainerPathStat{}, nil
		},
	}
	options := copyOptions{source: splitCpArg("container:/path"), destination: splitCpArg("-")}
	cli := test.NewFakeCli(fakeClient)
	err := runCopy(cli, options, fromContainer)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(tarContent, cli.OutBuffer().String()))
	assert.Check(t, is.Equal("", cli.ErrBuffer().String()))
}

func TestRunCopyFromContainerToFilesystem(t *testing.T) {
	destDir := fs.NewDir(t, "cp-test",
		fs.WithFile("file1", "content\n"))
	defer destDir.Remove()

	fakeClient := &fakeClient{
		containerCopyFromFunc: func(container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
			assert.Check(t, is.Equal("container", container))
			readCloser, err := archive.TarWithOptions(destDir.Path(), &archive.TarOptions{})
			return readCloser, types.ContainerPathStat{}, err
		},
	}
	options := copyOptions{source: splitCpArg("container:/path"), destination: splitCpArg(destDir.Path())}
	cli := test.NewFakeCli(fakeClient)
	err := runCopy(cli, options, fromContainer)
	assert.NilError(t, err)
	assert.Check(t, is.Equal("", cli.OutBuffer().String()))
	assert.Check(t, is.Equal("", cli.ErrBuffer().String()))

	content, err := ioutil.ReadFile(destDir.Join("file1"))
	assert.NilError(t, err)
	assert.Check(t, is.Equal("content\n", string(content)))
}

func TestRunCopyFromContainerToFilesystemMissingDestinationDirectory(t *testing.T) {
	destDir := fs.NewDir(t, "cp-test",
		fs.WithFile("file1", "content\n"))
	defer destDir.Remove()

	fakeClient := &fakeClient{
		containerCopyFromFunc: func(container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
			assert.Check(t, is.Equal("container", container))
			readCloser, err := archive.TarWithOptions(destDir.Path(), &archive.TarOptions{})
			return readCloser, types.ContainerPathStat{}, err
		},
	}

	options := copyOptions{
		source:      splitCpArg("container:/path"),
		destination: splitCpArg(destDir.Join("missing", "foo")),
	}
	cli := test.NewFakeCli(fakeClient)
	err := runCopy(cli, options, fromContainer)
	assert.ErrorContains(t, err, destDir.Join("missing"))
}

func TestRunCopyToContainerFromFileWithTrailingSlash(t *testing.T) {
	srcFile := fs.NewFile(t, t.Name())
	defer srcFile.Remove()

	options := copyOptions{
		source:      splitCpArg(srcFile.Path() + string(os.PathSeparator)),
		destination: splitCpArg("container:/path"),
	}
	cli := test.NewFakeCli(&fakeClient{})
	err := runCopy(cli, options, toContainer)

	expectedError := "not a directory"
	if runtime.GOOS == "windows" {
		expectedError = "The filename, directory name, or volume label syntax is incorrect"
	}
	assert.ErrorContains(t, err, expectedError)
}

func TestRunCopyToContainerSourceDoesNotExist(t *testing.T) {
	options := copyOptions{
		source:      splitCpArg("/does/not/exist"),
		destination: splitCpArg("container:/path"),
	}
	cli := test.NewFakeCli(&fakeClient{})
	err := runCopy(cli, options, toContainer)
	expected := "no such file or directory"
	if runtime.GOOS == "windows" {
		expected = "cannot find the file specified"
	}
	assert.ErrorContains(t, err, expected)
}

func TestSplitCpArg(t *testing.T) {
	var testcases = []struct {
		doc               string
		path              string
		os                string
		expectedContainer string
		expectedPath      string
	}{
		{
			doc:          "absolute path with colon",
			os:           "linux",
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
	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			skip.If(t, testcase.os != "" && testcase.os != runtime.GOOS)

			containerWithPath := splitCpArg(testcase.path)
			assert.Check(t, is.Equal(testcase.expectedContainer, containerWithPath.container))
			assert.Check(t, is.Equal(testcase.expectedPath, containerWithPath.path))
		})
	}
}
