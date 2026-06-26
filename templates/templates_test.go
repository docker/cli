package templates

import (
	"bytes"
	"testing"
	"text/template"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// GitHub #32120
func TestParseJSONFunctions(t *testing.T) {
	tm, err := Parse(`{{json .Ports}}`)
	assert.NilError(t, err)

	var b bytes.Buffer
	assert.NilError(t, tm.Execute(&b, map[string]string{"Ports": "0.0.0.0:2->8/udp"}))
	want := "\"0.0.0.0:2->8/udp\""
	assert.Check(t, is.Equal(want, b.String()))
}

func TestParseStringFunctions(t *testing.T) {
	tm, err := Parse(`{{join (split . ":") "/"}}`)
	assert.NilError(t, err)

	var b bytes.Buffer
	assert.NilError(t, tm.Execute(&b, "text:with:colon"))
	want := "text/with/colon"
	assert.Check(t, is.Equal(want, b.String()))
}

func TestNewParse(t *testing.T) {
	tm, err := New("foo").Parse("this is a {{ . }}")
	assert.NilError(t, err)

	var b bytes.Buffer
	assert.NilError(t, tm.Execute(&b, "string"))
	want := "this is a string"
	assert.Check(t, is.Equal(want, b.String()))
}

func TestParseTruncateFunction(t *testing.T) {
	source := "tupx5xzf6hvsrhnruz5cr8gwp"

	testCases := []struct {
		template string
		expected string
	}{
		{
			template: `{{truncate . 5}}`,
			expected: "tupx5",
		},
		{
			template: `{{truncate . 25}}`,
			expected: "tupx5xzf6hvsrhnruz5cr8gwp",
		},
		{
			template: `{{truncate . 30}}`,
			expected: "tupx5xzf6hvsrhnruz5cr8gwp",
		},
		{
			template: `{{pad . 3 3}}`,
			expected: "   tupx5xzf6hvsrhnruz5cr8gwp   ",
		},
	}

	for _, tc := range testCases {
		tm, err := Parse(tc.template)
		assert.NilError(t, err)

		t.Run("Non Empty Source Test with template: "+tc.template, func(t *testing.T) {
			var b bytes.Buffer
			assert.NilError(t, tm.Execute(&b, source))
			assert.Check(t, is.Equal(tc.expected, b.String()))
		})

		t.Run("Empty Source Test with template: "+tc.template, func(t *testing.T) {
			var c bytes.Buffer
			assert.NilError(t, tm.Execute(&c, ""))
			assert.Check(t, is.Equal("", c.String()))
		})

		t.Run("Nil Source Test with template: "+tc.template, func(t *testing.T) {
			var c bytes.Buffer
			assert.Check(t, tm.Execute(&c, nil) != nil)
			assert.Check(t, is.Equal("", c.String()))
		})
	}
}

func TestHeaderFunctions(t *testing.T) {
	const source = "hello world"

	tests := []struct {
		doc      string
		template string
	}{
		{
			doc:      "json",
			template: `{{ json .}}`,
		},
		{
			doc:      "split",
			template: `{{ split . ","}}`,
		},
		{
			doc:      "join",
			template: `{{ join . ","}}`,
		},
		{
			doc:      "title",
			template: `{{ title .}}`,
		},
		{
			doc:      "lower",
			template: `{{ lower .}}`,
		},
		{
			doc:      "upper",
			template: `{{ upper .}}`,
		},
		{
			doc:      "truncate",
			template: `{{ truncate . 2}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			tmpl, err := New("").Funcs(HeaderFunctions).Parse(tc.template)
			assert.NilError(t, err)

			var b bytes.Buffer
			assert.NilError(t, tmpl.Execute(&b, source))

			// All header-functions are currently stubs, and don't modify the input.
			expected := source
			assert.Equal(t, expected, b.String())
		})
	}
}

type stringerString string

func (s stringerString) String() string {
	return "stringer" + string(s)
}

type stringerAndError string

func (s stringerAndError) String() string {
	return "stringer" + string(s)
}

func (s stringerAndError) Error() string {
	return "error" + string(s)
}

func TestJoinElements(t *testing.T) {
	tests := []struct {
		doc    string
		data   any
		expOut string
		expErr string
	}{
		{
			doc:    "nil",
			data:   nil,
			expOut: `output: ""`,
		},
		{
			doc:    "non-slice",
			data:   "hello",
			expOut: `output: "`,
			expErr: `error calling join: expected slice, got string`,
		},
		{
			doc:    "structs",
			data:   []struct{ A, B string }{{"1", "2"}, {"3", "4"}},
			expOut: `output: "{1 2}, {3 4}"`,
		},
		{
			doc:    "map with strings",
			data:   map[string]string{"A": "1", "B": "2", "C": "3"},
			expOut: `output: "1, 2, 3"`,
		},
		{
			doc:    "map with stringers",
			data:   map[string]stringerString{"A": "1", "B": "2", "C": "3"},
			expOut: `output: "stringer1, stringer2, stringer3"`,
		},
		{
			doc:    "map with errors",
			data:   []stringerAndError{"1", "2", "3"},
			expOut: `output: "error1, error2, error3"`,
		},
		{
			doc:    "stringers",
			data:   []stringerString{"1", "2", "3"},
			expOut: `output: "stringer1, stringer2, stringer3"`,
		},
		{
			doc:    "stringer with errors",
			data:   []stringerAndError{"1", "2", "3"},
			expOut: `output: "error1, error2, error3"`,
		},
		{
			doc:    "slice of bools",
			data:   []bool{true, false, true},
			expOut: `output: "true, false, true"`,
		},
	}

	const formatStr = `output: "{{- join . ", " -}}"`
	tmpl, err := New("my-template").Funcs(template.FuncMap{"join": joinElements}).Parse(formatStr)
	assert.NilError(t, err)

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			var b bytes.Buffer
			err := tmpl.Execute(&b, tc.data)
			if tc.expErr != "" {
				assert.ErrorContains(t, err, tc.expErr)
			} else {
				assert.NilError(t, err)
			}
			assert.Equal(t, b.String(), tc.expOut)
		})
	}
}
