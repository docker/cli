package swarm

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestLoadBundlefileErrors(t *testing.T) {
	testCases := []struct {
		namespace     string
		path          string
		expectedError string
	}{
		{
			namespace:     "namespace_foo",
			expectedError: "Bundle namespace_foo.dab not found",
		},
		{
			namespace:     "namespace_foo",
			path:          "invalid_path",
			expectedError: "Bundle invalid_path not found",
		},
		// FIXME: this test never working, testdata file is missing from repo
		//{
		//	namespace:     "namespace_foo",
		//	path:          string(golden.Get(t, "bundlefile_with_invalid_syntax")),
		//	expectedError: "Error reading",
		//},
	}

	for _, tc := range testCases {
		_, err := loadBundlefile(&bytes.Buffer{}, tc.namespace, tc.path)
		assert.ErrorContains(t, err, tc.expectedError)
	}
}

func TestLoadBundlefile(t *testing.T) {
	buf := new(bytes.Buffer)

	namespace := ""
	path := filepath.Join("testdata", "bundlefile_with_two_services.dab")
	bundleFile, err := loadBundlefile(buf, namespace, path)

	assert.NilError(t, err)
	assert.Check(t, is.Equal(len(bundleFile.Services), 2))
}
