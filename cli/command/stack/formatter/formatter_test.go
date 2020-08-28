package formatter

import (
	"bytes"
	"testing"

	"github.com/docker/cli/cli/command/formatter"
	"gotest.tools/v3/assert"
)

func TestStackContextWrite(t *testing.T) {
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
			formatter.Context{Format: SwarmStackTableFormat},
			`NAME      SERVICES   ORCHESTRATOR
baz       2          orchestrator1
bar       1          orchestrator2
`,
		},
		// Kubernetes table format adds Namespace column
		{
			formatter.Context{Format: KubernetesStackTableFormat},
			`NAME      SERVICES   ORCHESTRATOR    NAMESPACE
baz       2          orchestrator1   namespace1
bar       1          orchestrator2   namespace2
`,
		},
		{
			formatter.Context{Format: formatter.Format("table {{.Name}}")},
			`NAME
baz
bar
`,
		},
		// Custom Format
		{
			formatter.Context{Format: formatter.Format("{{.Name}}")},
			`baz
bar
`,
		},
	}

	stacks := []*Stack{
		{Name: "baz", Services: 2, Orchestrator: "orchestrator1", Namespace: "namespace1"},
		{Name: "bar", Services: 1, Orchestrator: "orchestrator2", Namespace: "namespace2"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out

			if err := StackWrite(tc.context, stacks); err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}
