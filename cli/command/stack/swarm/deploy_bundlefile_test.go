package swarm

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestLoadBundlefileErrors(t *testing.T) {
	testCases := []struct {
		namespace     string
		path          string
		expectedError error
	}{
		{
			namespace:     "namespace_foo",
			expectedError: fmt.Errorf("Bundle %s.dab not found", "namespace_foo"),
		},
		{
			namespace:     "namespace_foo",
			path:          "invalid_path",
			expectedError: fmt.Errorf("Bundle %s not found", "invalid_path"),
		},
		{
			namespace:     "namespace_foo",
			path:          filepath.Join("testdata", "bundlefile_with_invalid_syntax"),
			expectedError: fmt.Errorf("Error reading"),
		},
	}

	for _, tc := range testCases {
		_, err := loadBundlefile(&bytes.Buffer{}, tc.namespace, tc.path)
		assert.Check(t, is.ErrorContains(err, ""), tc.expectedError)
	}
}

func TestLoadBundlefile(t *testing.T) {
	buf := new(bytes.Buffer)

	namespace := ""
	path := filepath.Join("testdata", "bundlefile_with_two_services.dab")
	bundleFile, err := loadBundlefile(buf, namespace, path)

	assert.Check(t, err)
	assert.Check(t, is.Equal(len(bundleFile.Services), 2))
}
