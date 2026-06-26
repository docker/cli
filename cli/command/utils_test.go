package command_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/command"
	"gotest.tools/v3/assert"
)

func TestValidateOutputPath(t *testing.T) {
	basedir := t.TempDir()
	dir := filepath.Join(basedir, "dir")
	notexist := filepath.Join(basedir, "notexist")
	err := os.MkdirAll(dir, 0o755)
	assert.NilError(t, err)
	file := filepath.Join(dir, "file")
	err = os.WriteFile(file, []byte("hi"), 0o644)
	assert.NilError(t, err)
	testcases := []struct {
		path string
		err  error
	}{
		{basedir, nil},
		{file, nil},
		{dir, nil},
		{dir + string(os.PathSeparator), nil},
		{notexist, nil},
		{notexist + string(os.PathSeparator), nil},
		{filepath.Join(notexist, "file"), errors.New("does not exist")},
	}

	for _, testcase := range testcases {
		t.Run(testcase.path, func(t *testing.T) {
			err := command.ValidateOutputPath(testcase.path)
			if testcase.err == nil {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, testcase.err.Error())
			}
		})
	}
}
