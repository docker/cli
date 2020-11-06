package registry

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/test"
	registrytypes "github.com/docker/docker/api/types/registry"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestSearchContext(t *testing.T) {
	name := "nginx"
	starCount := 5000

	var ctx searchContext
	cases := []struct {
		searchCtx searchContext
		expValue  string
		call      func() string
	}{
		{searchContext{
			s: registrytypes.SearchResult{Name: name},
		}, name, ctx.Name},
		{searchContext{
			s: registrytypes.SearchResult{StarCount: starCount},
		}, "5000", ctx.StarCount},
		{searchContext{
			s: registrytypes.SearchResult{IsOfficial: true},
		}, "[OK]", ctx.IsOfficial},
		{searchContext{
			s: registrytypes.SearchResult{IsOfficial: false},
		}, "", ctx.IsOfficial},
		{searchContext{
			s: registrytypes.SearchResult{IsAutomated: true},
		}, "[OK]", ctx.IsAutomated},
		{searchContext{
			s: registrytypes.SearchResult{IsAutomated: false},
		}, "", ctx.IsAutomated},
	}

	for _, c := range cases {
		ctx = c.searchCtx
		v := c.call()
		if strings.Contains(v, ",") {
			test.CompareMultipleValues(t, v, c.expValue)
		} else if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}
}

func TestSearchContextDescription(t *testing.T) {
	shortDescription := "Official build of Nginx."
	longDescription := "Automated Nginx reverse proxy for docker containers"
	descriptionWReturns := "Automated\nNginx reverse\rproxy\rfor docker\ncontainers"

	var ctx searchContext
	cases := []struct {
		searchCtx searchContext
		expValue  string
		call      func() string
	}{
		{searchContext{
			s:     registrytypes.SearchResult{Description: shortDescription},
			trunc: true,
		}, shortDescription, ctx.Description},
		{searchContext{
			s:     registrytypes.SearchResult{Description: shortDescription},
			trunc: false,
		}, shortDescription, ctx.Description},
		{searchContext{
			s:     registrytypes.SearchResult{Description: longDescription},
			trunc: false,
		}, longDescription, ctx.Description},
		{searchContext{
			s:     registrytypes.SearchResult{Description: longDescription},
			trunc: true,
		}, formatter.Ellipsis(longDescription, 45), ctx.Description},
		{searchContext{
			s:     registrytypes.SearchResult{Description: descriptionWReturns},
			trunc: false,
		}, longDescription, ctx.Description},
		{searchContext{
			s:     registrytypes.SearchResult{Description: descriptionWReturns},
			trunc: true,
		}, formatter.Ellipsis(longDescription, 45), ctx.Description},
	}

	for _, c := range cases {
		ctx = c.searchCtx
		v := c.call()
		if strings.Contains(v, ",") {
			test.CompareMultipleValues(t, v, c.expValue)
		} else if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}
}

func TestSearchContextWrite(t *testing.T) {
	cases := []struct {
		context  formatter.Context
		expected string
	}{

		// Errors
		{
			formatter.Context{Format: "{{InvalidFunction}}"},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			formatter.Context{Format: "{{nil}}"},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table format
		{
			formatter.Context{Format: NewSearchFormat("table")},
			string(golden.Get(t, "search-context-write-table.golden")),
		},
		{
			formatter.Context{Format: NewSearchFormat("table {{.Name}}")},
			`NAME
result1
result2
`,
		},
		// Custom Format
		{
			formatter.Context{Format: NewSearchFormat("{{.Name}}")},
			`result1
result2
`,
		},
		// Custom Format with CreatedAt
		{
			formatter.Context{Format: NewSearchFormat("{{.Name}} {{.StarCount}}")},
			`result1 5000
result2 5
`,
		},
	}

	results := []registrytypes.SearchResult{
		{Name: "result1", Description: "Official build", StarCount: 5000, IsOfficial: true, IsAutomated: false},
		{Name: "result2", Description: "Not official", StarCount: 5, IsOfficial: false, IsAutomated: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out

			err := SearchWrite(tc.context, results)
			if err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestSearchContextWriteJSON(t *testing.T) {
	results := []registrytypes.SearchResult{
		{Name: "result1", Description: "Official build", StarCount: 5000, IsOfficial: true, IsAutomated: false},
		{Name: "result2", Description: "Not official", StarCount: 5, IsOfficial: false, IsAutomated: true},
	}
	expectedJSONs := []map[string]interface{}{
		{"Name": "result1", "Description": "Official build", "StarCount": "5000", "IsOfficial": "true", "IsAutomated": "false"},
		{"Name": "result2", "Description": "Not official", "StarCount": "5", "IsOfficial": "false", "IsAutomated": "true"},
	}

	out := bytes.NewBufferString("")
	err := SearchWrite(formatter.Context{Format: "{{json .}}", Output: out}, results)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		t.Logf("Output: line %d: %s", i, line)
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.DeepEqual(m, expectedJSONs[i]))
	}
}

func TestSearchContextWriteJSONField(t *testing.T) {
	results := []registrytypes.SearchResult{
		{Name: "result1", Description: "Official build", StarCount: 5000, IsOfficial: true, IsAutomated: false},
		{Name: "result2", Description: "Not official", StarCount: 5, IsOfficial: false, IsAutomated: true},
	}
	out := bytes.NewBufferString("")
	err := SearchWrite(formatter.Context{Format: "{{json .Name}}", Output: out}, results)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		t.Logf("Output: line %d: %s", i, line)
		var s string
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.Equal(s, results[i].Name))
	}
}
