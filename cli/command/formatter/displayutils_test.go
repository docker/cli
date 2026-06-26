package formatter

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestTruncateID(t *testing.T) {
	tests := []struct {
		doc, id, expected string
	}{
		{
			doc:      "empty ID",
			id:       "",
			expected: "",
		},
		{
			// IDs are expected to be 12 (short) or 64 characters, and not be numeric only,
			// but TruncateID should handle these gracefully.
			doc:      "invalid ID",
			id:       "1234",
			expected: "1234",
		},
		{
			doc:      "full ID",
			id:       "90435eec5c4e124e741ef731e118be2fc799a68aba0466ec17717f24ce2ae6a2",
			expected: "90435eec5c4e",
		},
		{
			doc:      "digest",
			id:       "sha256:90435eec5c4e124e741ef731e118be2fc799a68aba0466ec17717f24ce2ae6a2",
			expected: "90435eec5c4e",
		},
		{
			doc:      "very long ID",
			id:       "90435eec5c4e124e741ef731e118be2fc799a68aba0466ec17717f24ce2ae6a290435eec5c4e124e741ef731e118be2fc799a68aba0466ec17717f24ce2ae6a2",
			expected: "90435eec5c4e",
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			actual := TruncateID(tc.id)
			if actual != tc.expected {
				t.Errorf("expected: %q, got: %q", tc.expected, actual)
			}
		})
	}
}

func TestEllipsis(t *testing.T) {
	testcases := []struct {
		source   string
		width    int
		expected string
	}{
		{source: "t🐳ststring", width: 0, expected: ""},
		{source: "t🐳ststring", width: 1, expected: "t"},
		{source: "t🐳ststring", width: 2, expected: "t…"},
		{source: "t🐳ststring", width: 6, expected: "t🐳st…"},
		{source: "t🐳ststring", width: 20, expected: "t🐳ststring"},
		{source: "你好世界teststring", width: 0, expected: ""},
		{source: "你好世界teststring", width: 1, expected: "你"},
		{source: "你好世界teststring", width: 3, expected: "你…"},
		{source: "你好世界teststring", width: 6, expected: "你好…"},
		{source: "你好世界teststring", width: 20, expected: "你好世界teststring"},
	}

	for _, testcase := range testcases {
		assert.Check(t, is.Equal(testcase.expected, Ellipsis(testcase.source, testcase.width)))
	}
}
