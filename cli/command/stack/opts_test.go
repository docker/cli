package stack

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.Error(t, err, tc.expectedError)
	}
}

func TestLoadBundlefile(t *testing.T) {
	buf := new(bytes.Buffer)

	namespace := ""
	path := filepath.Join("testdata", "bundlefile_with_two_services.dab")
	bundleFile, err := loadBundlefile(buf, namespace, path)

	assert.NoError(t, err)
	assert.Equal(t, len(bundleFile.Services), 2)
}
