package formatter

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestEllipsis(t *testing.T) {
	var testcases = []struct {
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
