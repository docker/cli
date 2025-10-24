package task

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
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
			`template parsing error: template: :1: function "InvalidFunction" not defined`,
		},
		{
			formatter.Context{Format: "{{nil}}"},
			`template parsing error: template: :1:2: executing "" at <nil>: nil is not a command`,
		},
		{
			formatter.Context{Format: newTaskFormat("table", true)},
			`taskID1
taskID2
`,
		},
		{
			formatter.Context{Format: newTaskFormat("table {{.Name}}\t{{.Node}}\t{{.Ports}}", false)},
			string(golden.Get(t, "task-context-write-table-custom.golden")),
		},
		{
			formatter.Context{Format: newTaskFormat("table {{.Name}}", true)},
			`NAME
foobar_baz
foobar_bar
`,
		},
		{
			formatter.Context{Format: newTaskFormat("raw", true)},
			`id: taskID1
id: taskID2
`,
		},
		{
			formatter.Context{Format: newTaskFormat("{{.Name}} {{.Node}}", false)},
			`foobar_baz foo1
foobar_bar foo2
`,
		},
	}

	tasks := client.TaskListResult{
		Items: []swarm.Task{
			{ID: "taskID1"},
			{ID: "taskID2"},
		},
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
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out

			if err := formatWrite(tc.context, tasks, names, nodes); err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestTaskContextWriteJSONField(t *testing.T) {
	tasks := client.TaskListResult{
		Items: []swarm.Task{
			{ID: "taskID1"},
			{ID: "taskID2"},
		},
	}
	names := map[string]string{
		"taskID1": "foobar_baz",
		"taskID2": "foobar_bar",
	}
	out := bytes.NewBufferString("")
	err := formatWrite(formatter.Context{Format: "{{json .ID}}", Output: out}, tasks, names, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var s string
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.Equal(tasks.Items[i].ID, s))
	}
}
