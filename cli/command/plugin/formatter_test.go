// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package plugin

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/plugin"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestPluginContext(t *testing.T) {
	pluginID := test.RandomID()

	var pCtx pluginContext
	tests := []struct {
		pluginCtx pluginContext
		expValue  string
		call      func() string
	}{
		{
			pluginCtx: pluginContext{
				p:     plugin.Plugin{ID: pluginID},
				trunc: false,
			},
			expValue: pluginID,
			call:     pCtx.ID,
		},
		{
			pluginCtx: pluginContext{
				p:     plugin.Plugin{ID: pluginID},
				trunc: true,
			},
			expValue: formatter.TruncateID(pluginID),
			call:     pCtx.ID,
		},
		{
			pluginCtx: pluginContext{
				p: plugin.Plugin{Name: "plugin_name"},
			},
			expValue: "plugin_name",
			call:     pCtx.Name,
		},
		{
			pluginCtx: pluginContext{
				p: plugin.Plugin{Config: plugin.Config{Description: "plugin_description"}},
			},
			expValue: "plugin_description",
			call:     pCtx.Description,
		},
	}

	for _, tc := range tests {
		pCtx = tc.pluginCtx
		v := tc.call()
		if strings.Contains(v, ",") {
			test.CompareMultipleValues(t, v, tc.expValue)
		} else if v != tc.expValue {
			t.Fatalf("Expected %s, was %s\n", tc.expValue, v)
		}
	}
}

func TestPluginContextWrite(t *testing.T) {
	tests := []struct {
		doc      string
		context  formatter.Context
		expected string
	}{
		{
			doc:      "invalid function",
			context:  formatter.Context{Format: "{{InvalidFunction}}"},
			expected: `template parsing error: template: :1: function "InvalidFunction" not defined`,
		},
		{
			doc:      "nil template",
			context:  formatter.Context{Format: "{{nil}}"},
			expected: `template parsing error: template: :1:2: executing "" at <nil>: nil is not a command`,
		},
		{
			doc:     "table format",
			context: formatter.Context{Format: newFormat("table", false)},
			expected: `ID          NAME         DESCRIPTION     ENABLED
pluginID1   foobar_baz   description 1   true
pluginID2   foobar_bar   description 2   false
`,
		},
		{
			doc:     "table format, quiet",
			context: formatter.Context{Format: newFormat("table", true)},
			expected: `pluginID1
pluginID2
`,
		},
		{
			doc:     "table format name col",
			context: formatter.Context{Format: newFormat("table {{.Name}}", false)},
			expected: `NAME
foobar_baz
foobar_bar
`,
		},
		{
			doc:     "table format name col, quiet",
			context: formatter.Context{Format: newFormat("table {{.Name}}", true)},
			expected: `NAME
foobar_baz
foobar_bar
`,
		},
		{
			doc:     "raw format",
			context: formatter.Context{Format: newFormat("raw", false)},
			expected: `plugin_id: pluginID1
name: foobar_baz
description: description 1
enabled: true

plugin_id: pluginID2
name: foobar_bar
description: description 2
enabled: false

`,
		},
		{
			doc:     "raw format, quiet",
			context: formatter.Context{Format: newFormat("raw", true)},
			expected: `plugin_id: pluginID1
plugin_id: pluginID2
`,
		},
		{
			doc:     "custom format",
			context: formatter.Context{Format: newFormat("{{.Name}}", false)},
			expected: `foobar_baz
foobar_bar
`,
		},
	}

	plugins := client.PluginListResult{
		Items: []plugin.Plugin{
			{ID: "pluginID1", Name: "foobar_baz", Config: plugin.Config{Description: "description 1"}, Enabled: true},
			{ID: "pluginID2", Name: "foobar_bar", Config: plugin.Config{Description: "description 2"}, Enabled: false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out

			err := formatWrite(tc.context, plugins)
			if err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestPluginContextWriteJSON(t *testing.T) {
	plugins := client.PluginListResult{
		Items: []plugin.Plugin{
			{ID: "pluginID1", Name: "foobar_baz"},
			{ID: "pluginID2", Name: "foobar_bar"},
		},
	}
	expectedJSONs := []map[string]any{
		{"Description": "", "Enabled": false, "ID": "pluginID1", "Name": "foobar_baz", "PluginReference": ""},
		{"Description": "", "Enabled": false, "ID": "pluginID2", "Name": "foobar_bar", "PluginReference": ""},
	}

	out := bytes.NewBufferString("")
	err := formatWrite(formatter.Context{Format: "{{json .}}", Output: out}, plugins)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.DeepEqual(expectedJSONs[i], m))
	}
}

func TestPluginContextWriteJSONField(t *testing.T) {
	plugins := client.PluginListResult{
		Items: []plugin.Plugin{
			{ID: "pluginID1", Name: "foobar_baz"},
			{ID: "pluginID2", Name: "foobar_bar"},
		},
	}
	out := bytes.NewBufferString("")
	err := formatWrite(formatter.Context{Format: "{{json .ID}}", Output: out}, plugins)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var s string
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.Equal(plugins.Items[i].ID, s))
	}
}
