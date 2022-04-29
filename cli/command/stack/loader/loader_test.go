package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"
)

func TestGetConfigDetails(t *testing.T) {
	content := `
version: "3.0"
services:
  foo:
    image: alpine:3.5
`
	file := fs.NewFile(t, "test-get-config-details", fs.WithContent(content))
	defer file.Remove()

	details, err := GetConfigDetails([]string{file.Path()}, nil)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(filepath.Dir(file.Path()), details.WorkingDir))
	assert.Assert(t, is.Len(details.ConfigFiles, 1))
	assert.Check(t, is.Equal("3.0", details.ConfigFiles[0].Config["version"]))
	assert.Check(t, is.Len(details.Environment, len(os.Environ())))
}

func TestGetConfigDetailsStdin(t *testing.T) {
	content := `
version: "3.0"
services:
  foo:
    image: alpine:3.5
`
	details, err := GetConfigDetails([]string{"-"}, strings.NewReader(content))
	assert.NilError(t, err)
	cwd, err := os.Getwd()
	assert.NilError(t, err)
	assert.Check(t, is.Equal(cwd, details.WorkingDir))
	assert.Assert(t, is.Len(details.ConfigFiles, 1))
	assert.Check(t, is.Equal("3.0", details.ConfigFiles[0].Config["version"]))
	assert.Check(t, is.Len(details.Environment, len(os.Environ())))
}
