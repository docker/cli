package formatter

import (
	"bytes"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestStackContextWrite(t *testing.T) {
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
			Context{Format: NewStackFormat("table")},
			`NAME                SERVICES            ORCHESTRATOR
baz                 2                   orchestrator1
bar                 1                   orchestrator2
`,
		},
		{
			Context{Format: NewStackFormat("table {{.Name}}")},
			`NAME
baz
bar
`,
		},
		// Custom Format
		{
			Context{Format: NewStackFormat("{{.Name}}")},
			`baz
bar
`,
		},
	}

	stacks := []*Stack{
		{Name: "baz", Services: 2, Orchestrator: "orchestrator1"},
		{Name: "bar", Services: 1, Orchestrator: "orchestrator2"},
	}
	for _, testcase := range cases {
		out := bytes.NewBufferString("")
		testcase.context.Output = out
		err := StackWrite(testcase.context, stacks)
		if err != nil {
			assert.Check(t, is.ErrorContains(err, testcase.expected))
		} else {
			assert.Check(t, is.Equal(out.String(), testcase.expected))
		}
	}
}
