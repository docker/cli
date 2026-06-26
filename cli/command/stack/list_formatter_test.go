package stack

import (
	"bytes"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"gotest.tools/v3/assert"
)

func TestStackContextWrite(t *testing.T) {
	tests := []struct {
		name     string
		format   formatter.Format
		expected string
	}{
		{
			name:     "invalid function",
			format:   `{{InvalidFunction}}`,
			expected: `template parsing error: template: :1: function "InvalidFunction" not defined`,
		},
		{
			name:     "invalid placeholder",
			format:   `{{nil}}`,
			expected: `template parsing error: template: :1:2: executing "" at <nil>: nil is not a command`,
		},
		{
			name:   "table format",
			format: stackTableFormat,
			expected: `NAME      SERVICES
baz       2
bar       1
`,
		},
		{
			name:   "custom table format",
			format: `table {{.Name}}`,
			expected: `NAME
baz
bar
`,
		},
		{
			name:   "custom format",
			format: `{{.Name}}`,
			expected: `baz
bar
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			fmtCtx := formatter.Context{
				Format: tc.format,
				Output: &out,
			}
			if err := stackWrite(fmtCtx, []stackSummary{
				{Name: "baz", Services: 2},
				{Name: "bar", Services: 1},
			}); err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}
