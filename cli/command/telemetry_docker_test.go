package command

import (
	"net/url"
	"testing"
	"testing/fstest"

	"gotest.tools/v3/assert"
)

func TestWslSocketPath(t *testing.T) {
	u, err := url.Parse("unix:////./c:/my/file/path")
	assert.NilError(t, err)

	// Ensure host is empty.
	assert.Equal(t, u.Host, "")

	// Use a filesystem where the WSL path exists.
	fs := fstest.MapFS{
		"mnt/c/my/file/path": {},
	}
	assert.Equal(t, wslSocketPath(u.Path, fs), "/mnt/c/my/file/path")

	// Use a filesystem where the WSL path doesn't exist.
	fs = fstest.MapFS{
		"my/file/path": {},
	}
	assert.Equal(t, wslSocketPath(u.Path, fs), "")
}
