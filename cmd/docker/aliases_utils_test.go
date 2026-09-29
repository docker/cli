package main

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestStringSliceReplaceAt(t *testing.T) {
	tests := []struct {
		name         string
		s            []string
		find         []string
		replace      []string
		requireIndex int
		expected     []string
		ok           bool
	}{
		{
			name:         "replace",
			s:            []string{"abc", "foo", "bar", "bax"},
			find:         []string{"foo", "bar"},
			replace:      []string{"baz"},
			requireIndex: -1,
			expected:     []string{"abc", "baz", "bax"},
			ok:           true,
		},
		{
			name:         "find longer than input",
			s:            []string{"foo"},
			find:         []string{"foo", "bar"},
			replace:      []string{"baz"},
			requireIndex: -1,
			expected:     []string{"foo"},
		},
		{
			name:         "wrong required index",
			s:            []string{"abc", "foo", "bar", "bax"},
			find:         []string{"foo", "bar"},
			replace:      []string{"baz"},
			requireIndex: 0,
			expected:     []string{"abc", "foo", "bar", "bax"},
		},
		{
			name:         "required index",
			s:            []string{"foo", "bar", "bax"},
			find:         []string{"foo", "bar"},
			replace:      []string{"baz"},
			requireIndex: 0,
			expected:     []string{"baz", "bax"},
			ok:           true,
		},
		{
			name:         "remove",
			s:            []string{"abc", "foo", "bar", "baz"},
			find:         []string{"foo", "bar"},
			requireIndex: -1,
			expected:     []string{"abc", "baz"},
			ok:           true,
		},
		{
			name:         "empty find",
			s:            []string{"foo"},
			replace:      []string{"baz"},
			requireIndex: -1,
			expected:     []string{"foo"},
		},
		{
			name:         "overlapping match",
			s:            []string{"a", "a", "b"},
			find:         []string{"a", "b"},
			replace:      []string{"c"},
			requireIndex: -1,
			expected:     []string{"a", "c"},
			ok:           true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := stringSliceReplaceAt(tc.s, tc.find, tc.replace, tc.requireIndex)
			assert.Equal(t, tc.ok, ok)
			assert.DeepEqual(t, tc.expected, out)
		})
	}
}
