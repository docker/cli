package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	details, err := getConfigDetails([]string{file.Path()}, nil)
	require.NoError(t, err)
	assert.Equal(t, filepath.Dir(file.Path()), details.WorkingDir)
	require.Len(t, details.ConfigFiles, 1)
	assert.Equal(t, "3.0", details.ConfigFiles[0].Config["version"])
	assert.Len(t, details.Environment, len(os.Environ()))
}

func TestGetConfigDetailsStdin(t *testing.T) {
	content := `
version: "3.0"
services:
  foo:
    image: alpine:3.5
`
	details, err := getConfigDetails([]string{"-"}, strings.NewReader(content))
	require.NoError(t, err)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, cwd, details.WorkingDir)
	require.Len(t, details.ConfigFiles, 1)
	assert.Equal(t, "3.0", details.ConfigFiles[0].Config["version"])
	assert.Len(t, details.Environment, len(os.Environ()))
}
