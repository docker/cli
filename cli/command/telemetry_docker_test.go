package command

import (
	"io/fs"
	"net/url"
	"testing"
	"testing/fstest"

	"gotest.tools/v3/assert"
)

func TestWslSocketPath(t *testing.T) {
	testCases := []struct {
		doc      string
		fs       fs.FS
		url      string
		expected string
	}{
		{
			doc: "filesystem where WSL path does not exist",
			fs: fstest.MapFS{
				"my/file/path": {},
			},
			url:      "unix:////./c:/my/file/path",
			expected: "",
		},
		{
			doc: "filesystem where WSL path exists",
			fs: fstest.MapFS{
				"mnt/c/my/file/path": {},
			},
			url:      "unix:////./c:/my/file/path",
			expected: "/mnt/c/my/file/path",
		},
		{
			doc: "filesystem where WSL path exists uppercase URL",
			fs: fstest.MapFS{
				"mnt/c/my/file/path": {},
			},
			url:      "unix:////./C:/my/file/path",
			expected: "/mnt/c/my/file/path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			u, err := url.Parse(tc.url)
			assert.NilError(t, err)
			// Ensure host is empty.
			assert.Equal(t, u.Host, "")

			result := wslSocketPath(u.Path, tc.fs)

			assert.Equal(t, result, tc.expected)
		})
	}
}
