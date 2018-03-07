package logs

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/pkg/errors"
)

func TestParseLogDetails(t *testing.T) {
	testCases := []struct {
		line     string
		expected map[string]string
		err      error
	}{
		{"key=value", map[string]string{"key": "value"}, nil},
		{"key1=value1,key2=value2", map[string]string{"key1": "value1", "key2": "value2"}, nil},
		{"key+with+spaces=value%3Dequals,asdf%2C=", map[string]string{"key with spaces": "value=equals", "asdf,": ""}, nil},
		{"key=,=nothing", map[string]string{"key": "", "": "nothing"}, nil},
		{"=", map[string]string{"": ""}, nil},
		{"errors", nil, errors.New("invalid details format")},
	}
	for _, testcase := range testCases {
		t.Run(testcase.line, func(t *testing.T) {
			actual, err := ParseLogDetails(testcase.line)
			if testcase.err != nil {
				assert.Error(t, err, testcase.err.Error())
				return
			}
			assert.Check(t, is.DeepEqual(testcase.expected, actual))
		})
	}
}
