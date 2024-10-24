package container

import (
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
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
		containerListFunc   func(container.ListOptions) ([]types.Container, error)
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
			containerListFunc: func(container.ListOptions) ([]types.Container, error) {
				return []types.Container{
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

func TestCompleteSecurityOpt(t *testing.T) {
	tests := []struct {
		toComplete          string
		expectedCompletions []string
		expectedDirective   cobra.ShellCompDirective
	}{
		{
			toComplete:          "",
			expectedCompletions: []string{"apparmor=", "label=", "no-new-privileges", "seccomp=", "systempaths=unconfined"},
			expectedDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
		{
			toComplete:          "apparmor=",
			expectedCompletions: []string{"apparmor="},
			expectedDirective:   cobra.ShellCompDirectiveNoSpace,
		},
		{
			toComplete:          "label=",
			expectedCompletions: []string{"label=disable", "label=level:", "label=role:", "label=type:", "label=user:"},
			expectedDirective:   cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
		},
		{
			toComplete: "s",
			// We do not filter matching completions but delegate this task to the shell script.
			expectedCompletions: []string{"apparmor=", "label=", "no-new-privileges", "seccomp=", "systempaths=unconfined"},
			expectedDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
		{
			toComplete:          "se",
			expectedCompletions: []string{"seccomp="},
			expectedDirective:   cobra.ShellCompDirectiveNoSpace,
		},
		{
			toComplete:          "seccomp=",
			expectedCompletions: []string{"seccomp=unconfined"},
			expectedDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
		{
			toComplete:          "sy",
			expectedCompletions: []string{"apparmor=", "label=", "no-new-privileges", "seccomp=", "systempaths=unconfined"},
			expectedDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tc := range tests {
		t.Run(tc.toComplete, func(t *testing.T) {
			completions, directive := completeSecurityOpt(nil, nil, tc.toComplete)
			assert.Check(t, is.DeepEqual(completions, tc.expectedCompletions))
			assert.Check(t, is.Equal(directive, tc.expectedDirective))
		})
	}
}

func TestCompleteSignals(t *testing.T) {
	values, directives := completeSignals(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Check(t, len(values) > 1)
	assert.Check(t, is.Len(values, len(signal.SignalMap)))
}
