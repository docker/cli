package build

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/fileutils"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

const dockerfileContents = "FROM busybox"

var prepareEmpty = func(t *testing.T) (string, func()) {
	return "", func() {}
}

var prepareNoFiles = func(t *testing.T) (string, func()) {
	return createTestTempDir(t, "", "builder-context-test")
}

var prepareOneFile = func(t *testing.T) (string, func()) {
	contextDir, cleanup := createTestTempDir(t, "", "builder-context-test")
	createTestTempFile(t, contextDir, DefaultDockerfileName, dockerfileContents, 0777)
	return contextDir, cleanup
}

func testValidateContextDirectory(t *testing.T, prepare func(t *testing.T) (string, func()), excludes []string) {
	contextDir, cleanup := prepare(t)
	defer cleanup()

	err := ValidateContextDirectory(contextDir, excludes)
	assert.NilError(t, err)
}

func TestGetContextFromLocalDirNoDockerfile(t *testing.T) {
	contextDir, cleanup := createTestTempDir(t, "", "builder-context-test")
	defer cleanup()

	_, _, err := GetContextFromLocalDir(contextDir, "")
	assert.ErrorContains(t, err, "Dockerfile")
}

func TestGetContextFromLocalDirNotExistingDir(t *testing.T) {
	contextDir, cleanup := createTestTempDir(t, "", "builder-context-test")
	defer cleanup()

	fakePath := filepath.Join(contextDir, "fake")

	_, _, err := GetContextFromLocalDir(fakePath, "")
	assert.ErrorContains(t, err, "fake")
}

func TestGetContextFromLocalDirNotExistingDockerfile(t *testing.T) {
	contextDir, cleanup := createTestTempDir(t, "", "builder-context-test")
	defer cleanup()

	fakePath := filepath.Join(contextDir, "fake")

	_, _, err := GetContextFromLocalDir(contextDir, fakePath)
	assert.ErrorContains(t, err, "fake")
}

func TestGetContextFromLocalDirWithNoDirectory(t *testing.T) {
	contextDir, dirCleanup := createTestTempDir(t, "", "builder-context-test")
	defer dirCleanup()

	createTestTempFile(t, contextDir, DefaultDockerfileName, dockerfileContents, 0777)

	chdirCleanup := chdir(t, contextDir)
	defer chdirCleanup()

	absContextDir, relDockerfile, err := GetContextFromLocalDir(contextDir, "")
	assert.NilError(t, err)

	assert.Check(t, is.Equal(contextDir, absContextDir))
	assert.Check(t, is.Equal(DefaultDockerfileName, relDockerfile))
}

func TestGetContextFromLocalDirWithDockerfile(t *testing.T) {
	contextDir, cleanup := createTestTempDir(t, "", "builder-context-test")
	defer cleanup()

	createTestTempFile(t, contextDir, DefaultDockerfileName, dockerfileContents, 0777)

	absContextDir, relDockerfile, err := GetContextFromLocalDir(contextDir, "")
	assert.NilError(t, err)

	assert.Check(t, is.Equal(contextDir, absContextDir))
	assert.Check(t, is.Equal(DefaultDockerfileName, relDockerfile))
}

func TestGetContextFromLocalDirLocalFile(t *testing.T) {
	contextDir, cleanup := createTestTempDir(t, "", "builder-context-test")
	defer cleanup()

	createTestTempFile(t, contextDir, DefaultDockerfileName, dockerfileContents, 0777)
	testFilename := createTestTempFile(t, contextDir, "tmpTest", "test", 0777)

	absContextDir, relDockerfile, err := GetContextFromLocalDir(testFilename, "")

	if err == nil {
		t.Fatalf("Error should not be nil")
	}

	if absContextDir != "" {
		t.Fatalf("Absolute directory path should be empty, got: %s", absContextDir)
	}

	if relDockerfile != "" {
		t.Fatalf("Relative path to Dockerfile should be empty, got: %s", relDockerfile)
	}
}

func TestGetContextFromLocalDirWithCustomDockerfile(t *testing.T) {
	contextDir, cleanup := createTestTempDir(t, "", "builder-context-test")
	defer cleanup()

	chdirCleanup := chdir(t, contextDir)
	defer chdirCleanup()

	createTestTempFile(t, contextDir, DefaultDockerfileName, dockerfileContents, 0777)

	absContextDir, relDockerfile, err := GetContextFromLocalDir(contextDir, DefaultDockerfileName)
	assert.NilError(t, err)

	assert.Check(t, is.Equal(contextDir, absContextDir))
	assert.Check(t, is.Equal(DefaultDockerfileName, relDockerfile))
}

func TestGetContextFromReaderString(t *testing.T) {
	tarArchive, relDockerfile, err := GetContextFromReader(ioutil.NopCloser(strings.NewReader(dockerfileContents)), "")

	if err != nil {
		t.Fatalf("Error when executing GetContextFromReader: %s", err)
	}

	tarReader := tar.NewReader(tarArchive)

	_, err = tarReader.Next()

	if err != nil {
		t.Fatalf("Error when reading tar archive: %s", err)
	}

	buff := new(bytes.Buffer)
	buff.ReadFrom(tarReader)
	contents := buff.String()

	_, err = tarReader.Next()

	if err != io.EOF {
		t.Fatalf("Tar stream too long: %s", err)
	}

	assert.NilError(t, tarArchive.Close())

	if dockerfileContents != contents {
		t.Fatalf("Uncompressed tar archive does not equal: %s, got: %s", dockerfileContents, contents)
	}

	if relDockerfile != DefaultDockerfileName {
		t.Fatalf("Relative path not equals %s, got: %s", DefaultDockerfileName, relDockerfile)
	}
}

func TestGetContextFromReaderTar(t *testing.T) {
	contextDir, cleanup := createTestTempDir(t, "", "builder-context-test")
	defer cleanup()

	createTestTempFile(t, contextDir, DefaultDockerfileName, dockerfileContents, 0777)

	tarStream, err := archive.Tar(contextDir, archive.Uncompressed)
	assert.NilError(t, err)

	tarArchive, relDockerfile, err := GetContextFromReader(tarStream, DefaultDockerfileName)
	assert.NilError(t, err)

	tarReader := tar.NewReader(tarArchive)

	header, err := tarReader.Next()
	assert.NilError(t, err)

	if header.Name != DefaultDockerfileName {
		t.Fatalf("Dockerfile name should be: %s, got: %s", DefaultDockerfileName, header.Name)
	}

	buff := new(bytes.Buffer)
	buff.ReadFrom(tarReader)
	contents := buff.String()

	_, err = tarReader.Next()

	if err != io.EOF {
		t.Fatalf("Tar stream too long: %s", err)
	}

	assert.NilError(t, tarArchive.Close())

	if dockerfileContents != contents {
		t.Fatalf("Uncompressed tar archive does not equal: %s, got: %s", dockerfileContents, contents)
	}

	if relDockerfile != DefaultDockerfileName {
		t.Fatalf("Relative path not equals %s, got: %s", DefaultDockerfileName, relDockerfile)
	}
}

func TestValidateContextDirectoryEmptyContext(t *testing.T) {
	// This isn't a valid test on Windows. See https://play.golang.org/p/RR6z6jxR81.
	// The test will ultimately end up calling filepath.Abs(""). On Windows,
	// golang will error. On Linux, golang will return /. Due to there being
	// drive letters on Windows, this is probably the correct behaviour for
	// Windows.
	if runtime.GOOS == "windows" {
		t.Skip("Invalid test on Windows")
	}
	testValidateContextDirectory(t, prepareEmpty, []string{})
}

func TestValidateContextDirectoryContextWithNoFiles(t *testing.T) {
	testValidateContextDirectory(t, prepareNoFiles, []string{})
}

func TestValidateContextDirectoryWithOneFile(t *testing.T) {
	testValidateContextDirectory(t, prepareOneFile, []string{})
}

func TestValidateContextDirectoryWithOneFileExcludes(t *testing.T) {
	testValidateContextDirectory(t, prepareOneFile, []string{DefaultDockerfileName})
}

// createTestTempDir creates a temporary directory for testing.
// It returns the created path and a cleanup function which is meant to be used as deferred call.
// When an error occurs, it terminates the test.
func createTestTempDir(t *testing.T, dir, prefix string) (string, func()) {
	path, err := ioutil.TempDir(dir, prefix)
	assert.NilError(t, err)
	return path, func() { assert.NilError(t, os.RemoveAll(path)) }
}

// createTestTempFile creates a temporary file within dir with specific contents and permissions.
// When an error occurs, it terminates the test
func createTestTempFile(t *testing.T, dir, filename, contents string, perm os.FileMode) string {
	filePath := filepath.Join(dir, filename)
	err := ioutil.WriteFile(filePath, []byte(contents), perm)
	assert.NilError(t, err)
	return filePath
}

// chdir changes current working directory to dir.
// It returns a function which changes working directory back to the previous one.
// This function is meant to be executed as a deferred call.
// When an error occurs, it terminates the test.
func chdir(t *testing.T, dir string) func() {
	workingDirectory, err := os.Getwd()
	assert.NilError(t, err)
	assert.NilError(t, os.Chdir(dir))
	return func() { assert.NilError(t, os.Chdir(workingDirectory)) }
}

func TestIsArchive(t *testing.T) {
	var testcases = []struct {
		doc      string
		header   []byte
		expected bool
	}{
		{
			doc:      "nil is not a valid header",
			header:   nil,
			expected: false,
		},
		{
			doc:      "invalid header bytes",
			header:   []byte{0x00, 0x01, 0x02},
			expected: false,
		},
		{
			doc:      "header for bzip2 archive",
			header:   []byte{0x42, 0x5A, 0x68},
			expected: true,
		},
		{
			doc:      "header for 7zip archive is not supported",
			header:   []byte{0x50, 0x4b, 0x03, 0x04},
			expected: false,
		},
	}
	for _, testcase := range testcases {
		assert.Check(t, is.Equal(testcase.expected, IsArchive(testcase.header)), testcase.doc)
	}
}

func TestDetectArchiveReader(t *testing.T) {
	var testcases = []struct {
		file     string
		desc     string
		expected bool
	}{
		{
			file:     "../testdata/tar.test",
			desc:     "tar file without pax headers",
			expected: true,
		},
		{
			file:     "../testdata/gittar.test",
			desc:     "tar file with pax headers",
			expected: true,
		},
		{
			file:     "../testdata/Dockerfile.test",
			desc:     "not a tar file",
			expected: false,
		},
	}
	for _, testcase := range testcases {
		content, err := os.Open(testcase.file)
		assert.NilError(t, err)
		defer content.Close()

		_, isArchive, err := DetectArchiveReader(content)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(testcase.expected, isArchive), testcase.file)
	}
}

func mustPatternMatcher(t *testing.T, patterns []string) *fileutils.PatternMatcher {
	t.Helper()
	pm, err := fileutils.NewPatternMatcher(patterns)
	if err != nil {
		t.Fatal("failed to construct pattern matcher: ", err)
	}
	return pm
}

func TestWildcardMatches(t *testing.T) {
	match, _ := filepathMatches(mustPatternMatcher(t, []string{"*"}), "fileutils.go")
	if !match {
		t.Errorf("failed to get a wildcard match, got %v", match)
	}
}

// A simple pattern match should return true.
func TestPatternMatches(t *testing.T) {
	match, _ := filepathMatches(mustPatternMatcher(t, []string{"*.go"}), "fileutils.go")
	if !match {
		t.Errorf("failed to get a match, got %v", match)
	}
}

// An exclusion followed by an inclusion should return true.
func TestExclusionPatternMatchesPatternBefore(t *testing.T) {
	match, _ := filepathMatches(mustPatternMatcher(t, []string{"!fileutils.go", "*.go"}), "fileutils.go")
	if !match {
		t.Errorf("failed to get true match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderExclusions(t *testing.T) {
	match, _ := filepathMatches(mustPatternMatcher(t, []string{"docs", "!docs/README.md"}), "docs/README.md")
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderWithSlashExclusions(t *testing.T) {
	match, _ := filepathMatches(mustPatternMatcher(t, []string{"docs/", "!docs/README.md"}), "docs/README.md")
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderWildcardExclusions(t *testing.T) {
	match, _ := filepathMatches(mustPatternMatcher(t, []string{"docs/*", "!docs/README.md"}), "docs/README.md")
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A pattern followed by an exclusion should return false.
func TestExclusionPatternMatchesPatternAfter(t *testing.T) {
	match, _ := filepathMatches(mustPatternMatcher(t, []string{"*.go", "!fileutils.go"}), "fileutils.go")
	if match {
		t.Errorf("failed to get false match on exclusion pattern, got %v", match)
	}
}

// A filename evaluating to . should return false.
func TestExclusionPatternMatchesWholeDirectory(t *testing.T) {
	match, _ := filepathMatches(mustPatternMatcher(t, []string{"*.go"}), ".")
	if match {
		t.Errorf("failed to get false match on ., got %v", match)
	}
}

// Matches with no patterns
func TestMatchesWithNoPatterns(t *testing.T) {
	matches, err := filepathMatches(mustPatternMatcher(t, []string{}), "/any/path/there")
	if err != nil {
		t.Fatal(err)
	}
	if matches {
		t.Fatalf("Should not have match anything")
	}
}
