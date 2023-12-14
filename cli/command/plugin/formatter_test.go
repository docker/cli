// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package plugin

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestPluginContext(t *testing.T) {
	pluginID := stringid.GenerateRandomID()

	var ctx pluginContext
	cases := []struct {
		pluginCtx pluginContext
		expValue  string
		call      func() string
	}{
		{pluginContext{
			p:     types.Plugin{ID: pluginID},
			trunc: false,
		}, pluginID, ctx.ID},
		{pluginContext{
			p:     types.Plugin{ID: pluginID},
			trunc: true,
		}, stringid.TruncateID(pluginID), ctx.ID},
		{pluginContext{
			p: types.Plugin{Name: "plugin_name"},
		}, "plugin_name", ctx.Name},
		{pluginContext{
			p: types.Plugin{Config: types.PluginConfig{Description: "plugin_description"}},
		}, "plugin_description", ctx.Description},
	}

	for _, c := range cases {
		ctx = c.pluginCtx
		v := c.call()
		if strings.Contains(v, ",") {
			test.CompareMultipleValues(t, v, c.expValue)
		} else if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}
}

func TestPluginContextWrite(t *testing.T) {
	cases := []struct {
		context  formatter.Context
		expected string
	}{
		// Errors
		{
			formatter.Context{Format: "{{InvalidFunction}}"},
			`template parsing error: template: :1: function "InvalidFunction" not defined`,
		},
		{
			formatter.Context{Format: "{{nil}}"},
			`template parsing error: template: :1:2: executing "" at <nil>: nil is not a command`,
		},
		// Table format
		{
			formatter.Context{Format: NewFormat("table", false)},
			`ID          NAME         DESCRIPTION     ENABLED
pluginID1   foobar_baz   description 1   true
pluginID2   foobar_bar   description 2   false
`,
		},
		{
			formatter.Context{Format: NewFormat("table", true)},
			`pluginID1
pluginID2
`,
		},
		{
			formatter.Context{Format: NewFormat("table {{.Name}}", false)},
			`NAME
foobar_baz
foobar_bar
`,
		},
		{
			formatter.Context{Format: NewFormat("table {{.Name}}", true)},
			`NAME
foobar_baz
foobar_bar
`,
		},
		// Raw Format
		{
			formatter.Context{Format: NewFormat("raw", false)},
			`plugin_id: pluginID1
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
			formatter.Context{Format: NewFormat("raw", true)},
			`plugin_id: pluginID1
plugin_id: pluginID2
`,
		},
		// Custom Format
		{
			formatter.Context{Format: NewFormat("{{.Name}}", false)},
			`foobar_baz
foobar_bar
`,
		},
	}

	plugins := []*types.Plugin{
		{ID: "pluginID1", Name: "foobar_baz", Config: types.PluginConfig{Description: "description 1"}, Enabled: true},
		{ID: "pluginID2", Name: "foobar_bar", Config: types.PluginConfig{Description: "description 2"}, Enabled: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out

			err := FormatWrite(tc.context, plugins)
			if err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestPluginContextWriteJSON(t *testing.T) {
	plugins := []*types.Plugin{
		{ID: "pluginID1", Name: "foobar_baz"},
		{ID: "pluginID2", Name: "foobar_bar"},
	}
	expectedJSONs := []map[string]interface{}{
		{"Description": "", "Enabled": false, "ID": "pluginID1", "Name": "foobar_baz", "PluginReference": ""},
		{"Description": "", "Enabled": false, "ID": "pluginID2", "Name": "foobar_bar", "PluginReference": ""},
	}

	out := bytes.NewBufferString("")
	err := FormatWrite(formatter.Context{Format: "{{json .}}", Output: out}, plugins)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.DeepEqual(expectedJSONs[i], m))
	}
}

func TestPluginContextWriteJSONField(t *testing.T) {
	plugins := []*types.Plugin{
		{ID: "pluginID1", Name: "foobar_baz"},
		{ID: "pluginID2", Name: "foobar_bar"},
	}
	out := bytes.NewBufferString("")
	err := FormatWrite(formatter.Context{Format: "{{json .ID}}", Output: out}, plugins)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var s string
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.Equal(plugins[i].ID, s))
	}
}
