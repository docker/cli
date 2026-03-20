package container

import (
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
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
		containerListFunc   func(client.ContainerListOptions) (client.ContainerListResult, error)
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
			containerListFunc: func(client.ContainerListOptions) (client.ContainerListResult, error) {
				return client.ContainerListResult{
					Items: []container.Summary{
						*builders.Container("c1"),
						*builders.Container("c2"),
					},
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
			completions, directive := completePid(cli)(newRunCommand(cli), nil, tc.toComplete)
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

func TestCompleteLinks(t *testing.T) {
	tests := []struct {
		doc              string
		showAll, showIDs bool
		filters          []func(container.Summary) bool
		containers       []container.Summary
		expOut           []string
		expDirective     cobra.ShellCompDirective
	}{
		{
			doc:          "no results",
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			doc:     "all containers",
			showAll: true,
			containers: []container.Summary{
				{ID: "id-c", State: container.StateRunning, Names: []string{"/container-c", "/container-c/link-b", "/container-c/link-c"}},
				{ID: "id-b", State: container.StateCreated, Names: []string{"/container-b", "/container-b/link-a"}},
				{ID: "id-a", State: container.StateExited, Names: []string{"/container-a"}},
			},
			expOut:       []string{"container-c/link-b", "container-c/link-c", "container-b/link-a"},
			expDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			comp := completeLinks(test.NewFakeCli(&fakeClient{
				containerListFunc: func(client.ContainerListOptions) (client.ContainerListResult, error) {
					return client.ContainerListResult{Items: tc.containers}, nil
				},
			}))

			containers, directives := comp(&cobra.Command{}, nil, "")
			assert.Check(t, is.Equal(directives&tc.expDirective, tc.expDirective))
			assert.Check(t, is.DeepEqual(containers, tc.expOut))
		})
	}
}
