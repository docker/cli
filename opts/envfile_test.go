package opts

import (
	"os"
	"path/filepath"
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

// Test ParseEnvFile for a non existent file
func TestParseEnvFileNonExistentFile(t *testing.T) {
	_, err := ParseEnvFile("no_such_file")
	assert.Check(t, is.ErrorType(err, os.IsNotExist))
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
