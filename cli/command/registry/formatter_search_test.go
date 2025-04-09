package registry

import (
	"bytes"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	registrytypes "github.com/docker/docker/api/types/registry"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestSearchContext(t *testing.T) {
	var ctx searchContext
	cases := []struct {
		searchCtx searchContext
		expValue  string
		call      func() string
	}{
		{
			searchCtx: searchContext{
				s: registrytypes.SearchResult{Name: "nginx"},
			},
			expValue: "nginx",
			call:     ctx.Name,
		},
		{
			searchCtx: searchContext{
				s: registrytypes.SearchResult{StarCount: 5000},
			},
			expValue: "5000",
			call:     ctx.StarCount,
		},
		{
			searchCtx: searchContext{
				s: registrytypes.SearchResult{IsOfficial: true},
			},
			expValue: "[OK]",
			call:     ctx.IsOfficial,
		},
		{
			searchCtx: searchContext{
				s: registrytypes.SearchResult{IsOfficial: false},
			},
			call: ctx.IsOfficial,
		},
		{
			searchCtx: searchContext{
				s: registrytypes.SearchResult{IsAutomated: true}, //nolint:nolintlint,staticcheck // ignore SA1019 (IsAutomated is deprecated).
			},
			expValue: "[OK]",
			call:     ctx.IsAutomated, //nolint:nolintlint,staticcheck // ignore SA1019 (IsAutomated is deprecated).
		},
		{
			searchCtx: searchContext{
				s: registrytypes.SearchResult{},
			},
			call: ctx.IsAutomated, //nolint:nolintlint,staticcheck // ignore SA1019 (IsAutomated is deprecated).
		},
	}

	for _, c := range cases {
		ctx = c.searchCtx
		v := c.call()
		assert.Check(t, is.Equal(v, c.expValue))
	}
}

func TestSearchContextDescription(t *testing.T) {
	const (
		shortDescription    = "Official build of Nginx."
		longDescription     = "Automated Nginx reverse proxy for docker containers"
		descriptionWReturns = "Automated\nNginx reverse\rproxy\rfor docker\ncontainers"
	)

	var ctx searchContext
	cases := []struct {
		searchCtx searchContext
		expValue  string
		call      func() string
	}{
		{
			searchCtx: searchContext{
				s:     registrytypes.SearchResult{Description: shortDescription},
				trunc: true,
			},
			expValue: shortDescription,
			call:     ctx.Description,
		},
		{
			searchCtx: searchContext{
				s:     registrytypes.SearchResult{Description: shortDescription},
				trunc: false,
			},
			expValue: shortDescription,
			call:     ctx.Description,
		},
		{
			searchCtx: searchContext{
				s:     registrytypes.SearchResult{Description: longDescription},
				trunc: false,
			},
			expValue: longDescription,
			call:     ctx.Description,
		},
		{
			searchCtx: searchContext{
				s:     registrytypes.SearchResult{Description: longDescription},
				trunc: true,
			},
			expValue: formatter.Ellipsis(longDescription, 45),
			call:     ctx.Description,
		},
		{
			searchCtx: searchContext{
				s:     registrytypes.SearchResult{Description: descriptionWReturns},
				trunc: false,
			},
			expValue: longDescription,
			call:     ctx.Description,
		},
		{
			searchCtx: searchContext{
				s:     registrytypes.SearchResult{Description: descriptionWReturns},
				trunc: true,
			},
			expValue: formatter.Ellipsis(longDescription, 45),
			call:     ctx.Description,
		},
	}

	for _, c := range cases {
		ctx = c.searchCtx
		v := c.call()
		assert.Check(t, is.Equal(v, c.expValue))
	}
}

func TestSearchContextWrite(t *testing.T) {
	cases := []struct {
		doc         string
		format      formatter.Format
		expected    string
		expectedErr string
	}{
		{
			doc:         "Errors",
			format:      "{{InvalidFunction}}",
			expectedErr: `template parsing error: template: :1: function "InvalidFunction" not defined`,
		},
		{
			doc:         "Nil format",
			format:      "{{nil}}",
			expectedErr: `template parsing error: template: :1:2: executing "" at <nil>: nil is not a command`,
		},
		{
			doc:    "JSON format",
			format: "{{json .}}",
			expected: `{"Description":"Official build","IsAutomated":"false","IsOfficial":"true","Name":"result1","StarCount":"5000"}
{"Description":"Not official","IsAutomated":"true","IsOfficial":"false","Name":"result2","StarCount":"5"}
`,
		},
		{
			doc:    "JSON format, single field",
			format: "{{json .Name}}",
			expected: `"result1"
"result2"
`,
		},
		{
			doc:      "Table format",
			format:   NewSearchFormat("table"),
			expected: string(golden.Get(t, "search-context-write-table.golden")),
		},
		{
			doc:    "Table format, single column",
			format: NewSearchFormat("table {{.Name}}"),
			expected: `NAME
result1
result2
`,
		},
		{
			doc:    "Custom format, single field",
			format: NewSearchFormat("{{.Name}}"),
			expected: `result1
result2
`,
		},
		{
			doc:    "Custom Format, two columns",
			format: NewSearchFormat("{{.Name}} {{.StarCount}}"),
			expected: `result1 5000
result2 5
`,
		},
	}

	results := []registrytypes.SearchResult{
		{Name: "result1", Description: "Official build", StarCount: 5000, IsOfficial: true},
		{Name: "result2", Description: "Not official", StarCount: 5, IsAutomated: true},
	}

	for _, tc := range cases {
		t.Run(tc.doc, func(t *testing.T) {
			var out bytes.Buffer
			err := SearchWrite(formatter.Context{Format: tc.format, Output: &out}, results)
			if tc.expectedErr != "" {
				assert.Check(t, is.Error(err, tc.expectedErr))
			} else {
				assert.Check(t, err)
			}
			assert.Check(t, is.Equal(out.String(), tc.expected))
		})
	}
}
