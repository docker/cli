package formatter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	clitypes "github.com/docker/cli/types"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestUpdateContextWrite(t *testing.T) {
	cases := []struct {
		context  Context
		expected string
	}{
		// Errors
		{
			Context{Format: "{{InvalidFunction}}"},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			Context{Format: "{{nil}}"},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table format
		{
			Context{Format: NewUpdatesFormat("table", false)},
			`TYPE                VERSION             NOTES
updateType1         version1            description 1
updateType2         version2            description 2
`,
		},
		{
			Context{Format: NewUpdatesFormat("table", true)},
			`version1
version2
`,
		},
		{
			Context{Format: NewUpdatesFormat("table {{.Version}}", false)},
			`VERSION
version1
version2
`,
		},
		{
			Context{Format: NewUpdatesFormat("table {{.Version}}", true)},
			`VERSION
version1
version2
`,
		},
		// Raw Format
		{
			Context{Format: NewUpdatesFormat("raw", false)},
			`update_version: version1
type: updateType1
notes: description 1

update_version: version2
type: updateType2
notes: description 2

`,
		},
		{
			Context{Format: NewUpdatesFormat("raw", true)},
			`update_version: version1
update_version: version2
`,
		},
		// Custom Format
		{
			Context{Format: NewUpdatesFormat("{{.Version}}", false)},
			`version1
version2
`,
		},
	}

	for _, testcase := range cases {
		updates := []clitypes.Update{
			{Type: "updateType1", Version: "version1", Notes: "description 1"},
			{Type: "updateType2", Version: "version2", Notes: "description 2"},
		}
		out := &bytes.Buffer{}
		testcase.context.Output = out
		err := UpdatesWrite(testcase.context, updates)
		if err != nil {
			assert.Error(t, err, testcase.expected)
		} else {
			assert.Check(t, is.Equal(testcase.expected, out.String()))
		}
	}
}

func TestUpdateContextWriteJSON(t *testing.T) {
	updates := []clitypes.Update{
		{Type: "updateType1", Version: "version1", Notes: "note1"},
		{Type: "updateType2", Version: "version2", Notes: "note2"},
	}
	expectedJSONs := []map[string]interface{}{
		{"Version": "version1", "Notes": "note1", "Type": "updateType1"},
		{"Version": "version2", "Notes": "note2", "Type": "updateType2"},
	}

	out := &bytes.Buffer{}
	err := UpdatesWrite(Context{Format: "{{json .}}", Output: out}, updates)
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

func TestUpdateContextWriteJSONField(t *testing.T) {
	updates := []clitypes.Update{
		{Type: "updateType1", Version: "version1"},
		{Type: "updateType2", Version: "version2"},
	}
	out := &bytes.Buffer{}
	err := UpdatesWrite(Context{Format: "{{json .Type}}", Output: out}, updates)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var s string
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.Equal(updates[i].Type, s))
	}
}
