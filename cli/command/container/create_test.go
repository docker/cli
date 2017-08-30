package container

import (
	"os"
	"testing"

	"io/ioutil"

	"github.com/docker/cli/internal/test/testutil"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIDFileNoOPWithNoFilename(t *testing.T) {
	file, err := newCIDFile("")
	require.NoError(t, err)
	assert.Equal(t, &cidFile{}, file)

	assert.NoError(t, file.Write("id"))
	assert.NoError(t, file.Close())
}

func TestNewCIDFileWhenFileAlreadyExists(t *testing.T) {
	tempfile := fs.NewFile(t, "test-cid-file")
	defer tempfile.Remove()

	_, err := newCIDFile(tempfile.Path())
	testutil.ErrorContains(t, err, "Container ID file found")
}

func TestCIDFileCloseWithNoWrite(t *testing.T) {
	tempdir := fs.NewDir(t, "test-cid-file")
	defer tempdir.Remove()

	path := tempdir.Join("cidfile")
	file, err := newCIDFile(path)
	require.NoError(t, err)
	assert.Equal(t, file.path, path)

	assert.NoError(t, file.Close())
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestCIDFileCloseWithWrite(t *testing.T) {
	tempdir := fs.NewDir(t, "test-cid-file")
	defer tempdir.Remove()

	path := tempdir.Join("cidfile")
	file, err := newCIDFile(path)
	require.NoError(t, err)

	content := "id"
	assert.NoError(t, file.Write(content))

	actual, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(actual))

	assert.NoError(t, file.Close())
	_, err = os.Stat(path)
	require.NoError(t, err)
}
