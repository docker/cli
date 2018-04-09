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
			Context{Format: Format(SwarmStackTableFormat)},
			`NAME                SERVICES            ORCHESTRATOR
baz                 2                   orchestrator1
bar                 1                   orchestrator2
`,
		},
		// Kubernetes table format adds Namespace column
		{
			Context{Format: Format(KubernetesStackTableFormat)},
			`NAME                SERVICES            ORCHESTRATOR        NAMESPACE
baz                 2                   orchestrator1       namespace1
bar                 1                   orchestrator2       namespace2
`,
		},
		{
			Context{Format: Format("table {{.Name}}")},
			`NAME
baz
bar
`,
		},
		// Custom Format
		{
			Context{Format: Format("{{.Name}}")},
			`baz
bar
`,
		},
	}

	stacks := []*Stack{
		{Name: "baz", Services: 2, Orchestrator: "orchestrator1", Namespace: "namespace1"},
		{Name: "bar", Services: 1, Orchestrator: "orchestrator2", Namespace: "namespace2"},
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
