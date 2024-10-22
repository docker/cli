package credentials

import (
	"os"
	"path"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDetectDefaultStore(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)

	t.Run("none available", func(t *testing.T) {
		const expected = ""
		assert.Equal(t, expected, DetectDefaultStore(""))
	})
	t.Run("custom helper", func(t *testing.T) {
		const expected = "my-custom-helper"
		assert.Equal(t, expected, DetectDefaultStore(expected))

		// Custom helper should be used even if the actual helper exists
		createFakeHelper(t, path.Join(tmpDir, remoteCredentialsPrefix+defaultHelper))
		assert.Equal(t, expected, DetectDefaultStore(expected))
	})
	t.Run("default", func(t *testing.T) {
		createFakeHelper(t, path.Join(tmpDir, remoteCredentialsPrefix+defaultHelper))
		expected := defaultHelper
		assert.Equal(t, expected, DetectDefaultStore(""))
	})

	// On Linux, the "pass" credentials helper requires both a "pass" binary
	// to be present and a "docker-credentials-pass" credentials helper to
	// be installed.
	t.Run("preferred helper", func(t *testing.T) {
		// Create the default helper as we need it for the fallback.
		createFakeHelper(t, path.Join(tmpDir, remoteCredentialsPrefix+defaultHelper))

		const testPreferredHelper = "preferred"
		overridePreferred = testPreferredHelper

		// Use preferred helper if both binaries exist.
		t.Run("success", func(t *testing.T) {
			createFakeHelper(t, path.Join(tmpDir, testPreferredHelper))
			createFakeHelper(t, path.Join(tmpDir, remoteCredentialsPrefix+testPreferredHelper))
			expected := testPreferredHelper
			assert.Equal(t, expected, DetectDefaultStore(""))
		})

		// Fall back to the default helper if the preferred credentials-helper isn't installed.
		t.Run("not installed", func(t *testing.T) {
			createFakeHelper(t, path.Join(tmpDir, remoteCredentialsPrefix+testPreferredHelper))
			expected := defaultHelper
			assert.Equal(t, expected, DetectDefaultStore(""))
		})

		// Similarly, fall back to the default helper if the preferred credentials-helper
		// is installed, but the helper binary isn't found.
		t.Run("missing helper", func(t *testing.T) {
			createFakeHelper(t, path.Join(tmpDir, testPreferredHelper))
			expected := defaultHelper
			assert.Equal(t, expected, DetectDefaultStore(""))
		})
	})
}

func createFakeHelper(t *testing.T, fileName string) {
	t.Helper()
	assert.NilError(t, os.WriteFile(fileName, []byte("I'm a credentials-helper executable (really!)"), 0o700))
	t.Cleanup(func() {
		assert.NilError(t, os.RemoveAll(fileName))
	})
}
