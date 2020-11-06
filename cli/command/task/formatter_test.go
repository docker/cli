package task

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestTaskContextWrite(t *testing.T) {
	cases := []struct {
		context  formatter.Context
		expected string
	}{
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
		{
			formatter.Context{Format: NewTaskFormat("table", true)},
			`taskID1
taskID2
`,
		},
		{
			formatter.Context{Format: NewTaskFormat("table {{.Name}}\t{{.Node}}\t{{.Ports}}", false)},
			string(golden.Get(t, "task-context-write-table-custom.golden")),
		},
		{
			formatter.Context{Format: NewTaskFormat("table {{.Name}}", true)},
			`NAME
foobar_baz
foobar_bar
`,
		},
		{
			formatter.Context{Format: NewTaskFormat("raw", true)},
			`id: taskID1
id: taskID2
`,
		},
		{
			formatter.Context{Format: NewTaskFormat("{{.Name}} {{.Node}}", false)},
			`foobar_baz foo1
foobar_bar foo2
`,
		},
	}

	tasks := []swarm.Task{
		{ID: "taskID1"},
		{ID: "taskID2"},
	}
	names := map[string]string{
		"taskID1": "foobar_baz",
		"taskID2": "foobar_bar",
	}
	nodes := map[string]string{
		"taskID1": "foo1",
		"taskID2": "foo2",
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out

			if err := FormatWrite(tc.context, tasks, names, nodes); err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestTaskContextWriteJSONField(t *testing.T) {
	tasks := []swarm.Task{
		{ID: "taskID1"},
		{ID: "taskID2"},
	}
	names := map[string]string{
		"taskID1": "foobar_baz",
		"taskID2": "foobar_bar",
	}
	out := bytes.NewBufferString("")
	err := FormatWrite(formatter.Context{Format: "{{json .ID}}", Output: out}, tasks, names, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var s string
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.Equal(tasks[i].ID, s))
	}
}
