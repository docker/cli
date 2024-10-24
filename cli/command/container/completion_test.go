package container

import (
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types/container"
	"github.com/moby/sys/signal"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCompleteLinuxCapabilityNames(t *testing.T) {
	names, directives := completeLinuxCapabilityNames(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Assert(t, len(names) > 1)
	assert.Check(t, names[0] == allCaps)
	for _, name := range names[1:] {
		assert.Check(t, strings.HasPrefix(name, "CAP_"))
		assert.Check(t, is.Equal(name, strings.ToUpper(name)), "Should be formatted uppercase")
	}
}

func TestCompletePid(t *testing.T) {
	tests := []struct {
		containerListFunc   func(container.ListOptions) ([]container.Summary, error)
		toComplete          string
		expectedCompletions []string
		expectedDirective   cobra.ShellCompDirective
	}{
		{
			toComplete:          "",
			expectedCompletions: []string{"container:", "host"},
			expectedDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
		{
			toComplete:          "c",
			expectedCompletions: []string{"container:"},
			expectedDirective:   cobra.ShellCompDirectiveNoSpace,
		},
		{
			containerListFunc: func(container.ListOptions) ([]container.Summary, error) {
				return []container.Summary{
					*builders.Container("c1"),
					*builders.Container("c2"),
				}, nil
			},
			toComplete:          "container:",
			expectedCompletions: []string{"container:c1", "container:c2"},
			expectedDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tc := range tests {
		t.Run(tc.toComplete, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				containerListFunc: tc.containerListFunc,
			})
			completions, directive := completePid(cli)(NewRunCommand(cli), nil, tc.toComplete)
			assert.Check(t, is.DeepEqual(completions, tc.expectedCompletions))
			assert.Check(t, is.Equal(directive, tc.expectedDirective))
		})
	}
}

func TestCompleteRestartPolicies(t *testing.T) {
	values, directives := completeRestartPolicies(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	expected := restartPolicies
	assert.Check(t, is.DeepEqual(values, expected))
}

func TestCompleteSignals(t *testing.T) {
	values, directives := completeSignals(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Check(t, len(values) > 1)
	assert.Check(t, is.Len(values, len(signal.SignalMap)))
}
