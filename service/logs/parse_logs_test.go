package logs

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestParseLogDetails(t *testing.T) {
	testCases := []struct {
		line        string
		expected    map[string]string
		expectedErr string
	}{
		{
			line:     "key=value",
			expected: map[string]string{"key": "value"},
		},
		{
			line:     "key1=value1,key2=value2",
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			line:     "key+with+spaces=value%3Dequals,asdf%2C=",
			expected: map[string]string{"key with spaces": "value=equals", "asdf,": ""},
		},
		{
			line:     "key=,key2=",
			expected: map[string]string{"key": "", "key2": ""},
		},
		{
			line:        "key=,=nothing",
			expectedErr: "invalid details format",
		},
		{
			line:        "=nothing",
			expectedErr: "invalid details format",
		},
		{
			line:        "=",
			expectedErr: "invalid details format",
		},
		{
			line:        "errors",
			expectedErr: "invalid details format",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.line, func(t *testing.T) {
			actual, err := ParseLogDetails(tc.line)
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			} else {
				assert.Check(t, err)
			}
			assert.Check(t, is.DeepEqual(tc.expected, actual))
		})
	}
}
