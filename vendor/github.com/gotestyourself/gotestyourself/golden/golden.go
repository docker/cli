/*Package golden provides tools for comparing large mutli-line strings.

Golden files are files in the ./testdata/ subdirectory of the package under test.
*/
package golden

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var flagUpdate = flag.Bool("test.update-golden", false, "update golden file")

// Get returns the golden file content
func Get(t require.TestingT, filename string) []byte {
	expected, err := ioutil.ReadFile(Path(filename))
	require.NoError(t, err)
	return expected
}

// Path returns the full path to a golden file
func Path(filename string) string {
	return filepath.Join("testdata", filename)
}

func update(t require.TestingT, filename string, actual []byte) {
	if *flagUpdate {
		err := ioutil.WriteFile(Path(filename), actual, 0644)
		require.NoError(t, err)
	}
}

// Assert compares the actual content to the expected content in the golden file.
// If `--update-golden` is set then the actual content is written to the golden
// file.
// Returns whether the assertion was successful (true) or not (false)
func Assert(t require.TestingT, actual string, filename string, msgAndArgs ...interface{}) bool {
	expected := Get(t, filename)
	update(t, filename, []byte(actual))

	if assert.ObjectsAreEqual(expected, []byte(actual)) {
		return true
	}

	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(expected)),
		B:        difflib.SplitLines(actual),
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  3,
	})
	require.NoError(t, err, msgAndArgs...)
	return assert.Fail(t, fmt.Sprintf("Not Equal: \n%s", diff), msgAndArgs...)
}

// AssertBytes compares the actual result to the expected result in the golden
// file. If `--update-golden` is set then the actual content is written to the
// golden file.
// Returns whether the assertion was successful (true) or not (false)
// nolint: lll
func AssertBytes(t require.TestingT, actual []byte, filename string, msgAndArgs ...interface{}) bool {
	expected := Get(t, filename)
	update(t, filename, actual)
	return assert.Equal(t, expected, actual, msgAndArgs...)
}
