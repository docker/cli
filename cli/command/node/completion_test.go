package node

import (
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TestCompleteNodePsFilters(t *testing.T) {
	tests := []struct {
		doc        string
		toComplete string
		expected   []string
		directive  cobra.ShellCompDirective
	}{
		{
			doc:        "no input offers the filter keys",
			toComplete: "",
			expected:   []string{"desired-state=", "id=", "label=", "name="},
			directive:  cobra.ShellCompDirectiveNoSpace,
		},
		{
			doc:        "desired-state values",
			toComplete: "desired-state=",
			expected:   []string{"desired-state=running", "desired-state=shutdown", "desired-state=accepted"},
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:        "label offers no values",
			toComplete: "label=",
			expected:   nil,
			directive:  cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:        "unknown key falls back to the filter keys",
			toComplete: "bogus=",
			expected:   []string{"desired-state=", "id=", "label=", "name="},
			directive:  cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{})
			completions, directive := completeNodePsFilters(cli)(newPsCommand(cli), nil, tc.toComplete)
			assert.DeepEqual(t, completions, tc.expected)
			assert.Equal(t, directive, tc.directive)
		})
	}
}
