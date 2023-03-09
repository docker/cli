package loader

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestBuildEnvironment(t *testing.T) {
	inputEnv := []string{
		"LEGIT_VAR=LEGIT_VALUE",
		"EMPTY_VARIABLE=",
	}

	if runtime.GOOS == "windows" {
		inputEnv = []string{
			"LEGIT_VAR=LEGIT_VALUE",

			// cmd.exe has some special environment variables which start with "=".
			// These should be ignored as they're only there for MS-DOS compatibility.
			"=ExitCode=00000041",
			"=ExitCodeAscii=A",
			`=C:=C:\some\dir`,
			`=D:=D:\some\different\dir`,
			`=X:=X:\`,
			`=::=::\`,

			"EMPTY_VARIABLE=",
		}
	}

	env, err := buildEnvironment(inputEnv)
	assert.NilError(t, err)

	assert.Check(t, is.Len(env, 2))
	assert.Check(t, is.Equal("LEGIT_VALUE", env["LEGIT_VAR"]))
	assert.Check(t, is.Equal("", env["EMPTY_VARIABLE"]))
}
