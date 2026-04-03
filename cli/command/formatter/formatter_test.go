// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.25

package formatter

import (
	"bytes"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		doc      string
		f        Format
		isJSON   bool
		isTable  bool
		template string
	}{
		{
			doc:      "json format",
			f:        "json",
			isJSON:   true,
			isTable:  false,
			template: JSONFormat,
		},
		{
			doc:      "empty table format (no template)",
			f:        "table",
			isJSON:   false,
			isTable:  true,
			template: "",
		},
		{
			doc:      "table with escaped tabs",
			f:        "table {{.Field}}\\t{{.Field2}}",
			isJSON:   false,
			isTable:  true,
			template: "{{.Field}}\t{{.Field2}}",
		},
		{
			doc:      "table with raw string",
			f:        `table {{.Field}}\t{{.Field2}}`,
			isJSON:   false,
			isTable:  true,
			template: "{{.Field}}\t{{.Field2}}",
		},
		{
			doc:      "other format",
			f:        "other",
			isJSON:   false,
			isTable:  false,
			template: "other",
		},
		{
			doc:      "other with spaces",
			f:        "   other   ",
			isJSON:   false,
			isTable:  false,
			template: "other",
		},
		{
			doc:      "other with newline preserved",
			f:        "   other\n   ",
			isJSON:   false,
			isTable:  false,
			template: "other\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			assert.Check(t, is.Equal(tc.f.IsJSON(), tc.isJSON))
			assert.Check(t, is.Equal(tc.f.IsTable(), tc.isTable))
			assert.Check(t, is.Equal(tc.f.templateString(), tc.template))
		})
	}
}

type fakeSubContext struct {
	Name string
}

func (fakeSubContext) FullHeader() any {
	return map[string]string{"Name": "NAME"}
}

func TestContext(t *testing.T) {
	testCases := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:   "json format",
			format: JSONFormatKey,
			expected: `{"Name":"test"}
`,
		},
		{
			name:   "table format",
			format: `table {{.Name}}`,
			expected: `NAME
test
`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			ctx := Context{
				Format: Format(tc.format),
				Output: buf,
			}
			subContext := fakeSubContext{Name: "test"}
			subFormat := func(f func(sub SubContext) error) error {
				return f(subContext)
			}
			err := ctx.Write(&subContext, subFormat)
			assert.NilError(t, err)
			assert.Equal(t, buf.String(), tc.expected)
		})
	}
}
