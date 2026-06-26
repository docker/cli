package kvfile

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// Test Parse for a non existent file.
func TestParseNonExistentFile(t *testing.T) {
	_, err := Parse("no_such_file", nil)
	assert.Check(t, is.ErrorType(err, os.IsNotExist))
}

// Test Parse from a file with a lookup function.
func TestParseWithLookup(t *testing.T) {
	content := `# comment=
VAR=VAR_VALUE
EMPTY_VAR=
UNDEFINED_VAR
DEFINED_VAR
`
	vars := map[string]string{
		"DEFINED_VAR": "defined-value",
	}
	lookupFn := func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	}

	fileName := filepath.Join(t.TempDir(), "envfile")
	err := os.WriteFile(fileName, []byte(content), 0o644)
	assert.NilError(t, err)

	variables, err := Parse(fileName, lookupFn)
	assert.NilError(t, err)

	expectedLines := []string{"VAR=VAR_VALUE", "EMPTY_VAR=", "DEFINED_VAR=defined-value"}
	assert.Check(t, is.DeepEqual(variables, expectedLines))
}

// Test ParseEnvFile for a file with a few well formatted lines
func TestParseFromReaderGoodFile(t *testing.T) {
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

	lines, err := ParseFromReader(strings.NewReader(content), nil)
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

// Test ParseFromReader for an empty file
func TestParseFromReaderEmptyFile(t *testing.T) {
	lines, err := ParseFromReader(strings.NewReader(""), nil)
	assert.NilError(t, err)
	assert.Check(t, is.Len(lines, 0))
}

// Test ParseFromReader for a badly formatted file
func TestParseFromReaderBadlyFormattedFile(t *testing.T) {
	content := `foo=bar
    f   =quux
`
	_, err := ParseFromReader(strings.NewReader(content), nil)
	const expectedMessage = "variable 'f   ' contains whitespaces"
	assert.Check(t, is.ErrorContains(err, expectedMessage))
}

// Test ParseFromReader for a file with a line exceeding bufio.MaxScanTokenSize
func TestParseFromReaderLineTooLongFile(t *testing.T) {
	content := "foo=" + strings.Repeat("a", bufio.MaxScanTokenSize+42)

	_, err := ParseFromReader(strings.NewReader(content), nil)
	const expectedMessage = "bufio.Scanner: token too long"
	assert.Check(t, is.ErrorContains(err, expectedMessage))
}

// ParseEnvFile with a random file, pass through
func TestParseFromReaderRandomFile(t *testing.T) {
	content := `first line
another invalid line`

	_, err := ParseFromReader(strings.NewReader(content), nil)
	const expectedMessage = "variable 'first line' contains whitespaces"
	assert.Check(t, is.ErrorContains(err, expectedMessage))
}

// Test ParseFromReader with a lookup function.
func TestParseFromReaderWithLookup(t *testing.T) {
	content := `# comment=
VAR=VAR_VALUE
EMPTY_VAR=
UNDEFINED_VAR
DEFINED_VAR
`
	vars := map[string]string{
		"DEFINED_VAR": "defined-value",
	}
	lookupFn := func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	}

	variables, err := ParseFromReader(strings.NewReader(content), lookupFn)
	assert.NilError(t, err)

	expectedLines := []string{"VAR=VAR_VALUE", "EMPTY_VAR=", "DEFINED_VAR=defined-value"}
	assert.Check(t, is.DeepEqual(variables, expectedLines))
}

// Test ParseFromReader with empty variable name
func TestParseFromReaderWithNoName(t *testing.T) {
	content := `# comment=
=blank variable names are an error case
`
	_, err := ParseFromReader(strings.NewReader(content), nil)
	const expectedMessage = "no variable name on line '=blank variable names are an error case'"
	assert.Check(t, is.ErrorContains(err, expectedMessage))
}
