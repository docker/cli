// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package formatter

import (
	"bytes"
	"testing"

	"gotest.tools/v3/assert"
)

func TestFormat(t *testing.T) {
	f := Format("json")
	assert.Assert(t, f.IsJSON())
	assert.Assert(t, !f.IsTable())

	f = Format("table")
	assert.Assert(t, !f.IsJSON())
	assert.Assert(t, f.IsTable())

	f = Format("other")
	assert.Assert(t, !f.IsJSON())
	assert.Assert(t, !f.IsTable())
}

type fakeSubContext struct {
	Name string
}

func (f fakeSubContext) FullHeader() any {
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
