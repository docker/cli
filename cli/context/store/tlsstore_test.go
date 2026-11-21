package store

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/containerd/errdefs"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/skip"
)

func TestTlsCreateUpdateGetRemove(t *testing.T) {
	testee := tlsStore{root: t.TempDir()}

	const contextName = "test-ctx"

	_, err := testee.getData(contextName, "test-ep", "test-data")
	assert.ErrorType(t, err, errdefs.IsNotFound)

	err = testee.createOrUpdate(contextName, "test-ep", "test-data", []byte("data"))
	assert.NilError(t, err)
	data, err := testee.getData(contextName, "test-ep", "test-data")
	assert.NilError(t, err)
	assert.Equal(t, string(data), "data")
	err = testee.createOrUpdate(contextName, "test-ep", "test-data", []byte("data2"))
	assert.NilError(t, err)
	data, err = testee.getData(contextName, "test-ep", "test-data")
	assert.NilError(t, err)
	assert.Equal(t, string(data), "data2")

	err = testee.removeEndpoint(contextName, "test-ep")
	assert.NilError(t, err)
	_, err = testee.getData(contextName, "test-ep", "test-data")
	assert.ErrorType(t, err, errdefs.IsNotFound)
}

func TestTlsCreateDirectoryPermissions(t *testing.T) {
	skip.If(t, runtime.GOOS == "windows", "unix permissions are not supported on Windows")

	configDir := filepath.Join(t.TempDir(), ".docker")
	testee := tlsStore{root: filepath.Join(configDir, "contexts", tlsDir)}

	const contextName = "test-ctx"
	assert.NilError(t, testee.createOrUpdate(contextName, "test-ep", "test-data", []byte("data")))

	// All directories, including the config-directory if it did not yet exist,
	// should only be accessible by the current user, matching the permissions
	// used when saving the config-file.
	for _, dir := range []string{
		configDir,
		filepath.Dir(testee.root),
		testee.root,
		testee.contextDir(contextName),
		testee.endpointDir(contextName, "test-ep"),
	} {
		fi, err := os.Stat(dir)
		assert.NilError(t, err)
		assert.Check(t, is.Equal(fi.Mode().Perm(), os.FileMode(0o700)), dir)
	}
}

func TestTlsListAndBatchRemove(t *testing.T) {
	testee := tlsStore{root: t.TempDir()}

	all := map[string]EndpointFiles{
		"ep1": {"f1", "f2", "f3"},
		"ep2": {"f1", "f2", "f3"},
		"ep3": {"f1", "f2", "f3"},
	}

	ep1ep2 := map[string]EndpointFiles{
		"ep1": {"f1", "f2", "f3"},
		"ep2": {"f1", "f2", "f3"},
	}

	const contextName = "test-ctx"
	for name, files := range all {
		for _, file := range files {
			err := testee.createOrUpdate(contextName, name, file, []byte("data"))
			assert.NilError(t, err)
		}
	}

	resAll, err := testee.listContextData(contextName)
	assert.NilError(t, err)
	assert.DeepEqual(t, resAll, all)

	err = testee.removeEndpoint(contextName, "ep3")
	assert.NilError(t, err)
	resEp1ep2, err := testee.listContextData(contextName)
	assert.NilError(t, err)
	assert.DeepEqual(t, resEp1ep2, ep1ep2)

	err = testee.remove(contextName)
	assert.NilError(t, err)
	resEmpty, err := testee.listContextData(contextName)
	assert.NilError(t, err)
	assert.DeepEqual(t, resEmpty, map[string]EndpointFiles{})
}
