package command

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestStringSliceReplaceAt(t *testing.T) {
	out, ok := StringSliceReplaceAt([]string{"abc", "foo", "bar", "bax"}, []string{"foo", "bar"}, []string{"baz"}, -1)
	assert.Assert(t, ok)
	assert.DeepEqual(t, []string{"abc", "baz", "bax"}, out)

	out, ok = StringSliceReplaceAt([]string{"foo"}, []string{"foo", "bar"}, []string{"baz"}, -1)
	assert.Assert(t, !ok)
	assert.DeepEqual(t, []string{"foo"}, out)

	out, ok = StringSliceReplaceAt([]string{"abc", "foo", "bar", "bax"}, []string{"foo", "bar"}, []string{"baz"}, 0)
	assert.Assert(t, !ok)
	assert.DeepEqual(t, []string{"abc", "foo", "bar", "bax"}, out)

	out, ok = StringSliceReplaceAt([]string{"foo", "bar", "bax"}, []string{"foo", "bar"}, []string{"baz"}, 0)
	assert.Assert(t, ok)
	assert.DeepEqual(t, []string{"baz", "bax"}, out)

	out, ok = StringSliceReplaceAt([]string{"abc", "foo", "bar", "baz"}, []string{"foo", "bar"}, nil, -1)
	assert.Assert(t, ok)
	assert.DeepEqual(t, []string{"abc", "baz"}, out)

	out, ok = StringSliceReplaceAt([]string{"foo"}, nil, []string{"baz"}, -1)
	assert.Assert(t, !ok)
	assert.DeepEqual(t, []string{"foo"}, out)
}

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
			err := ValidateOutputPath(testcase.path)
			if testcase.err == nil {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, testcase.err.Error())
			}
		})
	}
}
