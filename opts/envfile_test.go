package opts

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func tmpFileWithContent(t *testing.T, content string) string {
	t.Helper()
	fileName := filepath.Join(t.TempDir(), "envfile")
	err := os.WriteFile(fileName, []byte(content), 0o644)
	assert.NilError(t, err)
	return fileName
}

// Test ParseEnvFile for a file with a few well formatted lines
func TestParseEnvFileGoodFile(t *testing.T) {
	content := `foo=bar
    baz=quux
# comment

_foobar=foobaz
with.dots=working
and_underscore=working too
`
	// Adding a newline + a line with pure whitespace.
	// This is being done like this instead of the block above
	// because it's common for editors to trim trailing whitespace
	// from lines, which becomes annoying since that's the
	// exact thing we need to test.
	content += "\n    \t  "
	tmpFile := tmpFileWithContent(t, content)

	lines, err := ParseEnvFile(tmpFile)
	assert.NilError(t, err)

	expectedLines := []string{
		"foo=bar",
		"baz=quux",
		"_foobar=foobaz",
		"with.dots=working",
		"and_underscore=working too",
	}

	assert.Check(t, is.DeepEqual(lines, expectedLines))
}

// Test ParseEnvFile for an empty file
func TestParseEnvFileEmptyFile(t *testing.T) {
	tmpFile := tmpFileWithContent(t, "")

	lines, err := ParseEnvFile(tmpFile)
	assert.NilError(t, err)
	assert.Check(t, is.Len(lines, 0))
}

// Test ParseEnvFile for a non existent file
func TestParseEnvFileNonExistentFile(t *testing.T) {
	_, err := ParseEnvFile("no_such_file")
	assert.Check(t, is.ErrorType(err, os.IsNotExist))
}

// Test ParseEnvFile for a badly formatted file
func TestParseEnvFileBadlyFormattedFile(t *testing.T) {
	content := `foo=bar
    f   =quux
`
	tmpFile := tmpFileWithContent(t, content)

	_, err := ParseEnvFile(tmpFile)
	const expectedMessage = "variable 'f   ' contains whitespaces"
	assert.Check(t, is.ErrorContains(err, expectedMessage))
}

// Test ParseEnvFile for a file with a line exceeding bufio.MaxScanTokenSize
func TestParseEnvFileLineTooLongFile(t *testing.T) {
	content := "foo=" + strings.Repeat("a", bufio.MaxScanTokenSize+42)
	tmpFile := tmpFileWithContent(t, content)

	_, err := ParseEnvFile(tmpFile)
	const expectedMessage = "bufio.Scanner: token too long"
	assert.Check(t, is.ErrorContains(err, expectedMessage))
}

// ParseEnvFile with a random file, pass through
func TestParseEnvFileRandomFile(t *testing.T) {
	content := `first line
another invalid line`
	tmpFile := tmpFileWithContent(t, content)

	_, err := ParseEnvFile(tmpFile)
	const expectedMessage = "variable 'first line' contains whitespaces"
	assert.Check(t, is.ErrorContains(err, expectedMessage))
}

// ParseEnvFile with environment variable import definitions
func TestParseEnvVariableDefinitionsFile(t *testing.T) {
	content := `# comment=
UNDEFINED_VAR
DEFINED_VAR
`
	tmpFile := tmpFileWithContent(t, content)

	t.Setenv("DEFINED_VAR", "defined-value")
	variables, err := ParseEnvFile(tmpFile)
	assert.NilError(t, err)

	expectedLines := []string{"DEFINED_VAR=defined-value"}
	assert.Check(t, is.DeepEqual(variables, expectedLines))
}

// ParseEnvFile with empty variable name
func TestParseEnvVariableWithNoNameFile(t *testing.T) {
	content := `# comment=
=blank variable names are an error case
`
	tmpFile := tmpFileWithContent(t, content)

	_, err := ParseEnvFile(tmpFile)
	const expectedMessage = "no variable name on line '=blank variable names are an error case'"
	assert.Check(t, is.ErrorContains(err, expectedMessage))
}
